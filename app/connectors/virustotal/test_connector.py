import unittest
from unittest import mock

from connectors.virustotal.connector import VirusTotal


def reply(status=200, body=None):
    return mock.Mock(status_code=status, json=mock.Mock(return_value=body), text=str(body))


class TestVirusTotal(unittest.TestCase):
    @mock.patch("requests.request", return_value=reply(body={"data": {"attributes": {
        "reputation": -5, "last_analysis_stats": {"malicious": 12, "suspicious": 1}}}}))
    def test_url_report_sums_up_the_analysis(self, request):
        result = VirusTotal().execute({"api_key": "k"}, {"url": "http://a.b/"}, "url_report")

        self.assertEqual(result["malicious"], 12)
        self.assertEqual(result["reputation"], -5)
        self.assertTrue(result["found"])
        self.assertEqual(request.call_args.args[1], "https://www.virustotal.com/api/v3/urls/aHR0cDovL2EuYi8")

    @mock.patch("requests.request", return_value=reply(404, {"error": {"code": "NotFoundError"}}))
    def test_file_report_of_an_unknown_hash_is_not_found(self, _):
        result = VirusTotal().execute({"api_key": "k"}, {"hash": "abc"}, "file_report")

        self.assertFalse(result["found"])
        self.assertEqual(result["malicious"], 0)


if __name__ == "__main__":
    unittest.main()
