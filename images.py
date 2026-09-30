"""Optional local previews, fetched from Instagram CDNs without session headers."""

from http.client import HTTPException
from pathlib import Path
import tempfile
import time
from urllib.error import HTTPError, URLError
from urllib.parse import urlsplit
from urllib.request import HTTPRedirectHandler, ProxyHandler, Request, build_opener

MAX_IMAGE_BYTES = 4 * 1024 * 1024


class ImageError(Exception):
    """A preview error containing no credentials, URL query, or response body."""


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def validate_image_url(url):
    if not isinstance(url, str) or len(url) > 8192:
        raise ImageError("missing or invalid preview URL")
    try:
        parsed = urlsplit(url)
        host = (parsed.hostname or "").lower()
        valid = (parsed.scheme == "https" and parsed.port in (None, 443)
                 and not parsed.username and not parsed.password
                 and not parsed.fragment
                 and any(host == domain or host.endswith("." + domain)
                         for domain in ("cdninstagram.com", "fbcdn.net")))
    except ValueError:
        valid = False
    if not valid or any(ord(c) < 33 or ord(c) == 127 for c in url):
        raise ImageError("preview URL is not an allowed HTTPS Instagram CDN")


def fetch_image(url):
    validate_image_url(url)
    # No bearer/cookie/captured headers, redirects, or implicit proxy settings.
    opener = build_opener(ProxyHandler({}), NoRedirect())
    request = Request(url, headers={"User-Agent": "instagram-scraper-preview/1"})
    try:
        with opener.open(request, timeout=20) as response:
            data = response.read(MAX_IMAGE_BYTES + 1)
    except HTTPError as exc:
        raise ImageError(f"preview HTTP {exc.code}") from None
    except (URLError, OSError, ValueError, HTTPException):
        raise ImageError("preview download failed") from None
    if len(data) > MAX_IMAGE_BYTES:
        raise ImageError("preview exceeds 4 MiB")
    if data.startswith(b"\x89PNG\r\n\x1a\n"):
        return data, ".png"
    if data.startswith(b"\xff\xd8\xff"):
        return data, ".jpg"
    raise ImageError("preview is not a PNG or JPEG image")


def save_previews(result, output, thumbnail_limit=12, delay=1.0):
    """Attach relative local paths; failures leave the raw snapshot usable."""
    if not 0 <= thumbnail_limit <= 100 or delay < 1:
        raise ImageError("invalid preview limits")
    output = Path(output)
    folder = output.parent / (output.stem + ".media")
    targets = [(result["profile"], "avatar_url", "avatar_path", "avatar")]
    targets += [(reel, "thumbnail_url", "thumbnail_path", f"reel-{index:03d}")
                for index, reel in enumerate(result["ranked_reels"][:thumbnail_limit], 1)]
    errors = []
    for record, url_key, path_key, name in targets:
        url = record.get(url_key)
        if not url:
            continue
        try:
            validate_image_url(url)
            time.sleep(delay)
            data, extension = fetch_image(url)
            if folder.is_symlink():
                raise ImageError("preview directory must not be a symlink")
            folder.mkdir(parents=True, exist_ok=True, mode=0o700)
            temporary_path = None
            try:
                with tempfile.NamedTemporaryFile(dir=folder, prefix=".preview-", delete=False) as temporary:
                    temporary_path = Path(temporary.name)
                    temporary.write(data)
                temporary_path.replace(folder / (name + extension))
            finally:
                if temporary_path is not None:
                    temporary_path.unlink(missing_ok=True)
            destination = folder / (name + extension)
            record[path_key] = destination.relative_to(output.parent).as_posix()
        except (ImageError, OSError) as exc:
            reason = str(exc) if isinstance(exc, ImageError) else "cannot save local preview"
            errors.append(f"{name}: {reason}")
    return errors
