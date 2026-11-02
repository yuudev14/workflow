import unittest
from unittest import mock

from connectors.urlscan.connector import URLScan


def reply(status=200, body=None):
    return mock.Mock(status_code=status, json=mock.Mock(return_value=body), text=str(body))


@mock.patch("time.sleep")
class TestURLScan(unittest.TestCase):
    def test_scan_url_waits_for_the_verdict(self, _):
        replies = [
            reply(body={"uuid": "u1", "result": "https://urlscan.io/result/u1/"}),
            reply(404, {"message": "not done"}),
            reply(body={"verdicts": {"overall": {"malicious": True, "score": 100, "categories": ["phishing"]}}}),
        ]
        with mock.patch("requests.request", side_effect=replies) as request:
            result = URLScan().execute({"api_key": "k"}, {"url": "http://bad.example"}, "scan_url")

        self.assertTrue(result["malicious"])
        self.assertEqual(result["categories"], ["phishing"])
        self.assertEqual(request.call_args_list[0].kwargs["json"], {"url": "http://bad.example", "visibility": "unlisted"})

    def test_scan_url_stops_after_the_max_wait(self, _):
        replies = [reply(body={"uuid": "u1"})] + [reply(404, {})] * 3
        with mock.patch("requests.request", side_effect=replies), \
                mock.patch("time.monotonic", side_effect=[0, 1, 2, 99]):
            with self.assertRaises(TimeoutError):
                URLScan().execute({}, {"url": "http://x", "max_wait_seconds": "10"}, "scan_url")


if __name__ == "__main__":
    unittest.main()
