from http.client import IncompleteRead
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from images import ImageError, MAX_IMAGE_BYTES, fetch_image, save_previews, validate_image_url


class ImageTests(unittest.TestCase):
    def test_cdn_boundary(self):
        for url in ("http://cdninstagram.com/x", "https://evil.test/x",
                    "https://cdninstagram.com.evil.test/x", "https://user:secret@fbcdn.net/x",
                    "https://fbcdn.net:123/x", "https://127.0.0.1/x"):
            with self.assertRaises(ImageError):
                validate_image_url(url)
        validate_image_url("https://scontent.cdninstagram.com/avatar.jpg?version=1")

    def test_download_does_not_send_session_headers(self):
        with patch("images.build_opener") as build:
            response = build.return_value.open.return_value.__enter__.return_value
            response.read.return_value = b"\x89PNG\r\n\x1a\nFAKE_TEST_IMAGE"
            fetch_image("https://scontent.cdninstagram.com/image.png")
            request = build.return_value.open.call_args.args[0]
            self.assertIsNone(request.get_header("Authorization"))
            self.assertIsNone(request.get_header("Cookie"))
            response.read.assert_called_once_with(MAX_IMAGE_BYTES + 1)

    def test_image_bounds_and_format(self):
        with patch("images.build_opener") as build:
            response = build.return_value.open.return_value.__enter__.return_value
            for data in (b"<html>no</html>", b"x" * (MAX_IMAGE_BYTES + 1)):
                response.read.return_value = data
                with self.assertRaises(ImageError):
                    fetch_image("https://scontent.cdninstagram.com/image.png")

    def test_interrupted_download_preserves_raw_metrics(self):
        result = {"profile": {"avatar_url": "https://fbcdn.net/avatar.png", "follower_count": 42},
                  "ranked_reels": []}
        with tempfile.TemporaryDirectory() as directory:
            with patch("images.time.sleep"), patch("images.build_opener") as build:
                response = build.return_value.open.return_value.__enter__.return_value
                response.read.side_effect = IncompleteRead(b"partial image", 100)
                errors = save_previews(result, Path(directory) / "example.json")
            self.assertEqual(errors, ["avatar: preview download failed"])
            self.assertEqual(result["profile"]["follower_count"], 42)
            self.assertNotIn("avatar_path", result["profile"])
            self.assertEqual(list(Path(directory).iterdir()), [])

    def test_partial_failure_keeps_data_and_paths_are_relative(self):
        result = {"profile": {"avatar_url": "https://fbcdn.net/avatar.png", "follower_count": 42},
                  "ranked_reels": [{"thumbnail_url": "https://fbcdn.net/one.png"},
                                   {"thumbnail_url": "https://fbcdn.net/two.png"}]}
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "example.json"
            with patch("images.time.sleep"), patch("images.fetch_image", side_effect=[
                    (b"fakepng", ".png"), ImageError("preview HTTP 403")]):
                errors = save_previews(result, path, thumbnail_limit=1)
            self.assertEqual(result["profile"]["avatar_path"], "example.media/avatar.png")
            self.assertEqual(result["profile"]["follower_count"], 42)
            self.assertEqual(errors, ["reel-001: preview HTTP 403"])
            self.assertNotIn("thumbnail_path", result["ranked_reels"][0])
            self.assertNotIn("thumbnail_path", result["ranked_reels"][1])
            self.assertEqual((Path(directory) / result["profile"]["avatar_path"]).stat().st_mode & 0o777, 0o600)

    def test_partial_disk_write_cleans_private_temporary_file(self):
        original = tempfile.NamedTemporaryFile

        def failing_file(*args, **kwargs):
            temporary = original(*args, **kwargs)
            write = temporary.write

            def partial_write(data):
                write(data[:2])
                raise OSError("synthetic disk full")

            temporary.write = partial_write
            return temporary

        result = {"profile": {"avatar_url": "https://fbcdn.net/avatar.png"}, "ranked_reels": []}
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "example.json"
            with patch("images.time.sleep"), patch("images.fetch_image", return_value=(b"fakepng", ".png")), \
                    patch("images.tempfile.NamedTemporaryFile", side_effect=failing_file):
                errors = save_previews(result, path)
            self.assertEqual(errors, ["avatar: cannot save local preview"])
            self.assertEqual(list((Path(directory) / "example.media").iterdir()), [])
            self.assertNotIn("avatar_path", result["profile"])


if __name__ == "__main__":
    unittest.main()
