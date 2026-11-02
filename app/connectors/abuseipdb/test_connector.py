import unittest
from unittest import mock

from connectors.abuseipdb.connector import AbuseIPDB


def reply(status=200, body=None):
    return mock.Mock(status_code=status, json=mock.Mock(return_value=body), text=str(body))


class TestAbuseIPDB(unittest.TestCase):
    @mock.patch("requests.request", return_value=reply(body={"data": {"abuseConfidenceScore": 97}}))
    def test_check_ip_returns_the_data(self, request):
        result = AbuseIPDB().execute({"api_key": "k", "base_url": "http://fake"}, {"ip": " 1.2.3.4 "}, "check_ip")

        self.assertEqual(result["abuseConfidenceScore"], 97)
        request.assert_called_once_with(
            "GET", "http://fake/check", headers={"Key": "k", "Accept": "application/json"},
            params={"ipAddress": "1.2.3.4", "maxAgeInDays": 90}, timeout=30,
        )

    @mock.patch("requests.request", return_value=reply(401, {"errors": "bad key"}))
    def test_raises_on_an_http_error(self, _):
        with self.assertRaisesRegex(RuntimeError, "401"):
            AbuseIPDB().execute({"api_key": "x"}, {"ip": "1.2.3.4"}, "check_ip")


if __name__ == "__main__":
    unittest.main()
