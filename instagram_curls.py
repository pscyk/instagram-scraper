"""The three read-only Instagram requests used by lookup.py.

The adapter provides session headers and a restricted transport. This module
contains no account mutations, login flow, scoring, or analysis.
"""

from urllib.parse import urlencode

import instagram_credentials
import utils


def get(path, query=None):
    url = "https://i.instagram.com/api/v1/" + path
    if query:
        url += "?" + urlencode(query)
    headers = instagram_credentials.build_curl_headers()
    return utils.cmd(f"{headers} -X GET '{url}'")


def username_info(username):
    return get(f"users/{username}/usernameinfo/")


def user_feed(user_id, max_id=None):
    return get(f"feed/user/{user_id}/", {"max_id": max_id} if max_id else None)


def media_info(media_pk):
    return get(f"media/{media_pk}/info/")
