import smtplib
from email.message import EmailMessage

from connectors.core.connector import Connector


def recipients(value: str) -> list[str]:
    return [item.strip() for item in (value or "").split(",") if item.strip()]


class Email(Connector):
    def execute(self, configs: dict, params: dict, operation: str, *args, **kwargs):
        if operation != "send_email":
            raise ValueError(f"operation ({operation}) does not exist in Email")
        to = recipients(params["to"])
        if not to:
            raise ValueError("to has no address")
        message = EmailMessage()
        message["From"] = configs["from"]
        message["To"] = ", ".join(to)
        message["Subject"] = params["subject"]
        message.set_content(params.get("body") or "")
        with self._connect(configs) as smtp:
            smtp.send_message(message)
        return {"sent": True, "to": to}

    def health_check(self, configs: dict, params: dict, operation: str, *args, **kwargs):
        with self._connect(configs) as smtp:
            smtp.noop()
        return {"ok": True}

    @staticmethod
    def _connect(configs: dict) -> smtplib.SMTP:
        security = (configs.get("security") or "starttls").lower()
        host, port = configs["host"], int(configs.get("port") or (465 if security == "ssl" else 587))
        smtp = smtplib.SMTP_SSL(host, port, timeout=30) if security == "ssl" else smtplib.SMTP(host, port, timeout=30)
        if security == "starttls":
            smtp.starttls()
        if configs.get("username"):
            smtp.login(configs["username"], configs.get("password", ""))
        return smtp
