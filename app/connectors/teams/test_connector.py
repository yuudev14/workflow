import unittest
from unittest import mock

from connectors.teams.connector import Teams


class TestTeams(unittest.TestCase):
    @mock.patch("requests.request", return_value=mock.Mock(status_code=202, json=mock.Mock(side_effect=ValueError), text=""))
    def test_send_message_posts_an_adaptive_card(self, request):
        Teams().execute({"webhook_url": "http://hook"}, {"title": "Alert", "text": "hi"}, "send_message")

        card = request.call_args.kwargs["json"]["attachments"][0]["content"]
        self.assertEqual([block["text"] for block in card["body"]], ["Alert", "hi"])


if __name__ == "__main__":
    unittest.main()
