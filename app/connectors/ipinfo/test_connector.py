import unittest
from unittest import mock

from connectors.ipinfo.connector import IPinfo


class TestIPinfo(unittest.TestCase):
    @mock.patch("requests.request", return_value=mock.Mock(status_code=200, json=mock.Mock(return_value={"country": "US"})))
    def test_lookup_ip_sends_the_token(self, request):
        result = IPinfo().execute({"token": "t"}, {"ip": "8.8.8.8"}, "lookup_ip")

        self.assertEqual(result["country"], "US")
        self.assertEqual(request.call_args.args[1], "https://ipinfo.io/8.8.8.8")
        self.assertEqual(request.call_args.kwargs["headers"]["Authorization"], "Bearer t")


if __name__ == "__main__":
    unittest.main()
