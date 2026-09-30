# Illustrative input format only. These placeholders are not a working session.
# Keep your own capture in .secrets/zebra_curls.sh, never in this example file.
# lookup.py parses this format; do not source or execute captured shell scripts.
curl -H 'authorization: Bearer REPLACE_WITH_YOUR_OWN_SESSION' \
  -H 'user-agent: REPLACE_WITH_YOUR_CAPTURED_USER_AGENT' \
  'https://i.instagram.com/api/v1/users/example/usernameinfo/'
