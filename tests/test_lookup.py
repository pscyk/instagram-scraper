import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

from lookup import LookupError, Transport, collect_feed, decode, load_wrappers, lookup, media_summary, read_headers


class LookupTests(unittest.TestCase):
    def test_original_wrapper_keeps_token_off_process_arguments(self):
        transport = Transport(["authorization: Bearer TEST_ONLY"], delay=1)
        api = load_wrappers(transport)
        response = subprocess.CompletedProcess([], 0, '{"status":"ok"}\n200', '')
        with patch("lookup.subprocess.run", return_value=response) as run:
            self.assertEqual(decode(api.username_info("example"))["status"], "ok")
        self.assertNotIn("TEST_ONLY", " ".join(run.call_args.args[0]))
        self.assertIn("TEST_ONLY", run.call_args.kwargs["input"])
        self.assertEqual(run.call_args.args[0][1], "-q")

    def test_capture_is_parsed_without_executing_commands(self):
        with tempfile.TemporaryDirectory() as tmp:
            capture = Path(tmp) / "capture.sh"
            capture.write_text("touch unwanted\ncurl -H 'authorization: Bearer TEST_ONLY' 'https://i.instagram.com/'\n")
            self.assertEqual(read_headers(capture), ["authorization: Bearer TEST_ONLY"])

    def test_transport_rejects_mutations_hosts_and_shell_arguments(self):
        transport = Transport([], delay=1)
        commands = [
            "curl -X POST https://i.instagram.com/api/v1/friendships/create/1/",
            "curl -X GET https://example.com/api/v1/feed/user/1/",
            "curl -X GET https://i.instagram.com/api/v1/feed/user/1/ --output stolen",
        ]
        with patch("lookup.subprocess.run") as run:
            for command in commands:
                with self.assertRaises(LookupError):
                    transport.command(command)
            run.assert_not_called()

    def test_throttled_response_stops_and_does_not_expose_body(self):
        response = subprocess.CompletedProcess([], 0, '{"message":"SECRET"}\n429', '')
        with patch("lookup.subprocess.run", return_value=response) as run:
            with self.assertRaisesRegex(LookupError, "HTTP 429") as error:
                Transport([], 1).command("curl -X GET https://i.instagram.com/api/v1/feed/user/1/")
            self.assertNotIn("SECRET", str(error.exception))
            run.assert_called_once()

    def test_repeated_cursor_is_incomplete(self):
        class API:
            def user_feed(self, *args):
                return json.dumps({"items": [{"pk": 1}], "more_available": True, "next_max_id": "same"})
        items, complete = collect_feed(API(), 1, 10)
        self.assertEqual(len(items), 1)
        self.assertFalse(complete)

    def test_lookup_preserves_missing_counts_and_ranks_measured_plays(self):
        class API:
            def username_info(self, username):
                return json.dumps({"user": {"pk": 1, "username": username}})

            def user_feed(self, *args):
                return json.dumps({"items": [{"pk": n, "product_type": "clips"} for n in (1, 2, 3)], "more_available": False})

            def media_info(self, pk):
                return json.dumps({"items": [{"pk": pk, "play_count": {1: None, 2: 0, 3: 10}[pk]}]})
        result = lookup(API(), "example", 10)
        self.assertEqual([r["pk"] for r in result["ranked_reels"]], [3, 2, 1])
        self.assertEqual(result["reels_missing_play_count"], 1)
        self.assertTrue(result["feed_complete"])

    def test_capture_errors_do_not_include_secrets(self):
        with tempfile.TemporaryDirectory() as tmp:
            capture = Path(tmp) / "capture.sh"
            capture.write_text("curl -H 'authorization: Bearer SECRET_UNCLOSED")
            with self.assertRaises(LookupError) as error:
                read_headers(capture)
            self.assertNotIn("SECRET", str(error.exception))

    def test_multiline_capture(self):
        with tempfile.TemporaryDirectory() as tmp:
            capture = Path(tmp) / "capture.sh"
            capture.write_text("curl \\\n  -H 'authorization: Bearer TEST_ONLY' \\\n  'https://i.instagram.com/'\n")
            self.assertEqual(read_headers(capture), ["authorization: Bearer TEST_ONLY"])

    def test_delay_floor_cannot_be_disabled(self):
        for delay in (0, -1, 0.9, float("nan"), float("inf")):
            with self.assertRaises(LookupError):
                Transport([], delay)

    def test_requests_are_paced(self):
        response = subprocess.CompletedProcess([], 0, '{"status":"ok"}\n200', '')
        transport = Transport([])
        with patch("lookup.subprocess.run", return_value=response), patch("lookup.time.sleep") as sleep:
            for _ in range(2):
                transport.command("curl -X GET https://i.instagram.com/api/v1/feed/user/1/")
        sleep.assert_called_once_with(1.0)

    def test_cursor_is_url_encoded(self):
        api = load_wrappers(Transport([]))
        response = subprocess.CompletedProcess([], 0, '{}\n200', '')
        with patch("lookup.subprocess.run", return_value=response) as run:
            api.user_feed(1, "cursor with & and ' quote")
        self.assertIn("max_id=cursor+with+%26+and+%27+quote", run.call_args.args[0][-1])

    def test_optional_image_metadata_does_not_change_counters(self):
        for versions in (None, "unexpected", {"candidates": "unexpected"}, {"candidates": [None]}):
            result = media_summary({"pk": 1, "play_count": 0, "image_versions2": versions})
            self.assertEqual(result["play_count"], 0)
            self.assertIsNone(result["thumbnail_url"])
        result = media_summary({"pk": 1, "image_versions2": {"candidates": [
            {"url": "https://scontent.cdninstagram.com/synthetic.png"}]}})
        self.assertEqual(result["thumbnail_url"], "https://scontent.cdninstagram.com/synthetic.png")


if __name__ == "__main__":
    unittest.main()
