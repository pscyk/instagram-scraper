#!/usr/bin/env python3
"""Save raw profile and reel metrics using a captured Instagram session."""

import argparse
from datetime import datetime, timezone
import importlib.util
import json
import math
import os
from pathlib import Path
import re
import shlex
import subprocess
import sys
import time
import types
from urllib.parse import urlparse

from images import save_previews

ROOT = Path(__file__).resolve().parent
PROFILE_FIELDS = (
    "pk", "username", "full_name", "biography", "external_url", "follower_count",
    "following_count", "media_count", "is_private", "is_verified", "category",
    "profile_pic_url", "profile_pic_url_hd",
)
MEDIA_FIELDS = (
    "pk", "code", "taken_at", "play_count", "ig_play_count", "fb_play_count",
    "view_count", "like_count", "comment_count", "reshare_count",
    "media_repost_count", "premium_reaction_count", "video_duration",
)
READ_ROUTES = (
    r"/api/v1/users/[A-Za-z0-9_.]+/usernameinfo/",
    r"/api/v1/feed/user/[0-9]+/",
    r"/api/v1/media/[0-9_]+/info/",
)
MAX_CAPTURE_BYTES = 256 * 1024
MIN_REQUEST_DELAY = 1.0


class LookupError(Exception):
    """An error safe to show without exposing session-bearing responses."""


def read_headers(capture):
    """Parse, but never execute, the first active curl line in the capture."""
    with Path(capture).open("rb") as source:
        raw = source.read(MAX_CAPTURE_BYTES + 1)
    if len(raw) > MAX_CAPTURE_BYTES:
        raise LookupError("Capture exceeds the 256 KiB limit")
    try:
        content = raw.decode("utf-8").replace("\\\r\n", "").replace("\\\n", "")
    except UnicodeDecodeError:
        raise LookupError("Capture must contain UTF-8 text") from None
    for line in content.splitlines():
        if not line.strip().startswith("curl "):
            continue
        try:
            args = shlex.split(line)
        except ValueError:
            raise LookupError("Malformed curl capture; check its quoting") from None
        headers = [args[i + 1] for i, v in enumerate(args[:-1]) if v == "-H"]
        if len(headers) > 128 or any(
                len(h) > 8192 or any(ord(c) < 32 or ord(c) == 127 for c in h)
                for h in headers):
            raise LookupError("Capture contains invalid headers")
        if not any(h.lower().startswith("authorization: bearer ") for h in headers):
            raise LookupError("Capture has no bearer authorization header")
        return headers
    raise LookupError("Capture contains no active curl command")


class Transport:
    def __init__(self, headers, delay=1.0):
        if not math.isfinite(delay) or delay < MIN_REQUEST_DELAY:
            raise LookupError("Delay must be at least one second")
        self.headers = headers
        self.delay = delay
        self.calls = 0

    def curl_prefix(self, credential=None):
        return shlex.join(["curl"] + [x for h in self.headers for x in ("-H", h)])

    def command(self, command):
        """Keep headers off process arguments and allow only lookup GET routes."""
        args = shlex.split(command)
        if not args or args[0] != "curl":
            raise LookupError("Unexpected command")
        headers, url, method, i = [], None, None, 1
        while i < len(args):
            value = args[i]
            if value in ("-H", "-X") and i + 1 < len(args):
                if value == "-H":
                    headers.append(args[i + 1])
                else:
                    method = args[i + 1]
                i += 2
            elif value.startswith("https://") and url is None:
                url = value
                i += 1
            else:
                raise LookupError("Unexpected curl argument")
        self.validate_url(url, method)
        config = "\n".join("header = " + json.dumps(h) for h in headers)
        if self.calls:
            time.sleep(self.delay)
        self.calls += 1
        # Disable curlrc and redirects so a local config cannot leak session headers.
        result = subprocess.run(
            ["curl", "-q", "--silent", "--show-error", "--compressed",
             "--connect-timeout", "10", "--max-time", "30", "--proto", "=https",
             "--config", "-", "--write-out", "\n%{http_code}", url],
            input=config, text=True, capture_output=True, timeout=35,
        )
        if result.returncode:
            raise LookupError(f"curl transport failed (exit {result.returncode})")
        try:
            body, status = result.stdout.rsplit("\n", 1)
            status = int(status)
        except ValueError as exc:
            raise LookupError("Invalid transport response") from exc
        print(f"HTTP {status} {urlparse(url).path}", file=sys.stderr, flush=True)
        if not 200 <= status < 300:
            raise LookupError(f"HTTP {status}; lookup stopped without retries")
        return body

    @staticmethod
    def validate_url(url, method):
        parsed = urlparse(url or "")
        if (method != "GET" or parsed.scheme != "https"
                or parsed.netloc != "i.instagram.com"
                or not any(re.fullmatch(route, parsed.path) for route in READ_ROUTES)):
            raise LookupError("Request is outside the supported read-only lookup")


def load_wrappers(transport):
    """Bind the read-only request helpers to this captured-session transport."""
    credentials = types.ModuleType("instagram_credentials")
    credentials.get_next_credential = lambda static=True: {}
    credentials.build_curl_headers = transport.curl_prefix
    utils = types.ModuleType("utils")
    utils.cmd = transport.command
    for module in (credentials, utils):
        sys.modules[module.__name__] = module
    spec = importlib.util.spec_from_file_location("instagram_curls", ROOT / "instagram_curls.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def decode(raw):
    try:
        data = json.loads(raw)
    except ValueError as exc:
        raise LookupError("Instagram returned a non-JSON response") from exc
    if not isinstance(data, dict) or data.get("status") == "fail":
        raise LookupError("Instagram rejected the request; check the session in the app")
    return data


def collect_feed(api, user_id, max_pages):
    items, cursors, cursor = {}, set(), None
    for page in range(1, max_pages + 1):
        data = decode(api.user_feed(user_id, cursor))
        if not isinstance(data.get("items"), list):
            raise LookupError("Feed response is missing items")
        for item in data["items"]:
            items[str(item["pk"])] = item
        print(f"Feed page {page}: {len(items)} posts", file=sys.stderr, flush=True)
        if data.get("more_available") is False:
            return list(items.values()), True
        cursor = data.get("next_max_id")
        if not cursor or cursor in cursors:
            return list(items.values()), False
        cursors.add(cursor)
    return list(items.values()), False


def media_summary(item):
    summary = {key: item.get(key) for key in MEDIA_FIELDS}
    summary["caption"] = (item.get("caption") or {}).get("text")
    code = item.get("code")
    summary["url"] = f"https://www.instagram.com/reel/{code}/" if code else None
    taken = item.get("taken_at")
    summary["published_at_utc"] = datetime.fromtimestamp(taken, timezone.utc).isoformat() if taken else None
    versions = item.get("image_versions2")
    candidates = versions.get("candidates") if isinstance(versions, dict) else []
    if not isinstance(candidates, list):
        candidates = []
    summary["thumbnail_url"] = next(
        (candidate["url"] for candidate in candidates
         if isinstance(candidate, dict) and isinstance(candidate.get("url"), str)), None)
    return summary


def lookup(api, username, max_pages):
    user = decode(api.username_info(username)).get("user")
    if not isinstance(user, dict) or str(user.get("username", "")).lower() != username.lower():
        raise LookupError("Profile response does not match the requested username")
    items, complete = collect_feed(api, user["pk"], max_pages)
    reels = [item for item in items if item.get("product_type") == "clips"]
    details = []
    for index, reel in enumerate(reels, 1):
        data = decode(api.media_info(reel["pk"]))
        matches = [x for x in data.get("items", []) if str(x.get("pk")) == str(reel["pk"])]
        if not matches:
            raise LookupError("Media response does not match the requested reel")
        details.append(media_summary(matches[0]))
        print(f"Reel metrics {index}/{len(reels)}", file=sys.stderr, flush=True)
    # Missing counts stay null and sort after measured zero; never substitute likes.
    details.sort(key=lambda x: x["play_count"] if x["play_count"] is not None else -1, reverse=True)
    profile = {key: user.get(key) for key in PROFILE_FIELDS}
    hd_info = user.get("hd_profile_pic_url_info")
    image_urls = [hd_info.get("url") if isinstance(hd_info, dict) else None,
                  user.get("profile_pic_url_hd"), user.get("profile_pic_url")]
    profile["avatar_url"] = next((url for url in image_urls if isinstance(url, str) and url), None)
    return {
        "measured_at_utc": datetime.now(timezone.utc).isoformat(),
        "profile": profile,
        "feed_complete": complete, "posts_scanned": len(items),
        "reels_checked": len(details), "ranking_metric": "play_count",
        "reels_missing_play_count": sum(x["play_count"] is None for x in details),
        "ranked_reels": details,
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("username")
    parser.add_argument("--capture", type=Path, default=ROOT / ".secrets/zebra_curls.sh")
    parser.add_argument("--output", type=Path)
    parser.add_argument("--delay", type=float, default=1.0, help="Seconds between requests (minimum/default: 1)")
    parser.add_argument("--max-pages", type=int, default=100)
    parser.add_argument("--download-images", action="store_true", help="Save an avatar and reel thumbnails next to --output")
    parser.add_argument("--thumbnail-limit", type=int, default=12, help="Maximum saved reel thumbnails (0–100; default: 12)")
    args = parser.parse_args()
    if not re.fullmatch(r"[A-Za-z0-9_.]{1,30}", args.username):
        parser.error("Invalid Instagram username")
    if not math.isfinite(args.delay) or args.delay < MIN_REQUEST_DELAY or args.max_pages < 1:
        parser.error("Delay must be at least one second and max-pages must be positive")
    if not 0 <= args.thumbnail_limit <= 100:
        parser.error("thumbnail-limit must be between 0 and 100")
    if args.download_images and not args.output:
        parser.error("--download-images requires --output")
    os.umask(0o077)
    try:
        transport = Transport(read_headers(args.capture), args.delay)
        result = lookup(load_wrappers(transport), args.username, args.max_pages)
        if args.download_images:
            for error in save_previews(result, args.output, args.thumbnail_limit, args.delay):
                print(f"Image unavailable: {error}", file=sys.stderr)
        encoded = json.dumps(result, indent=2, ensure_ascii=False)
        if args.output:
            args.output.parent.mkdir(parents=True, exist_ok=True)
            args.output.write_text(encoded + "\n")
            print(f"Saved {args.output}", file=sys.stderr)
        else:
            print(encoded)
        if not result["feed_complete"] or result["reels_missing_play_count"]:
            print("Warning: incomplete coverage; ranking may be incomplete", file=sys.stderr)
    except (LookupError, OSError, subprocess.TimeoutExpired) as exc:
        print(f"Lookup failed: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
