import unittest
from unittest import mock

from connectors.slack.connector import Slack


class TestSlack(unittest.TestCase):
    @mock.patch("requests.request", return_value=mock.Mock(status_code=200, json=mock.Mock(side_effect=ValueError), text="ok"))
    def test_send_message_posts_to_the_webhook(self, request):
        result = Slack().execute({"webhook_url": "http://hook"}, {"text": "hi"}, "send_message")

        self.assertTrue(result["sent"])
        request.assert_called_once_with("POST", "http://hook", json={"text": "hi"}, timeout=30)


if __name__ == "__main__":
    unittest.main()
