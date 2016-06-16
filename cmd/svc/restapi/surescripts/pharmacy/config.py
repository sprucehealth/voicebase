"""
Putting these credentials in a python module is a very simplistic approach.
You may want to use environment variables or some other mechanism. Be our guest.
"""


import urllib
import os

smartyStreetsAuthId = os.environ['SMARTY_STREETS_AUTH_ID']
smartyStreetsAuthToken = os.environ['SMARTY_STREETS_AUTH_TOKEN']

AUTHENTICATION = {
    'auth-id': smartyStreetsAuthId,
    'auth-token': urllib.unquote(
        smartyStreetsAuthToken
    ),
}
