import unittest
from unittest import mock

from connectors.email.connector import Email


class TestEmail(unittest.TestCase):
    @mock.patch("smtplib.SMTP")
    def test_send_email_uses_starttls_and_login(self, smtp_class):
        smtp = smtp_class.return_value.__enter__.return_value = smtp_class.return_value
        configs = {"host": "mail", "username": "u", "password": "p", "from": "soc@x.io"}

        result = Email().execute(configs, {"to": "a@x.io, b@x.io,", "subject": "Hi", "body": "text"}, "send_email")

        self.assertEqual(result["to"], ["a@x.io", "b@x.io"])
        smtp_class.assert_called_once_with("mail", 587, timeout=30)
        smtp.starttls.assert_called_once()
        smtp.login.assert_called_once_with("u", "p")
        message = smtp.send_message.call_args.args[0]
        self.assertEqual(message["To"], "a@x.io, b@x.io")
        self.assertEqual(message["Subject"], "Hi")

    @mock.patch("smtplib.SMTP")
    def test_security_none_skips_tls_and_login(self, smtp_class):
        smtp = smtp_class.return_value.__enter__.return_value = smtp_class.return_value

        Email().execute({"host": "mailpit", "port": "1025", "security": "none", "from": "f@x.io"},
                        {"to": "a@x.io", "subject": "Hi"}, "send_email")

        smtp_class.assert_called_once_with("mailpit", 1025, timeout=30)
        smtp.starttls.assert_not_called()
        smtp.login.assert_not_called()


if __name__ == "__main__":
    unittest.main()
