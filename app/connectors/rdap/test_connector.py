import unittest
from unittest import mock

from connectors.rdap.connector import RDAP

BODY = {
    "status": ["active"],
    "events": [{"eventAction": "registration", "eventDate": "2020-01-01T00:00:00Z"},
               {"eventAction": "expiration", "eventDate": "2030-01-01T00:00:00Z"}],
    "entities": [{"roles": ["registrar"], "vcardArray": ["vcard", [["version", {}, "text", "4.0"], ["fn", {}, "text", "Example Registrar"]]]}],
    "nameservers": [{"ldhName": "NS1.EXAMPLE.COM"}],
}


class TestRDAP(unittest.TestCase):
    @mock.patch("requests.request", return_value=mock.Mock(status_code=200, json=mock.Mock(return_value=BODY)))
    def test_domain_lookup_reads_the_registrar_and_the_age(self, request):
        result = RDAP().execute({}, {"domain": "Example.COM"}, "domain_lookup")

        self.assertEqual(result["registrar"], "Example Registrar")
        self.assertEqual(result["created"], "2020-01-01T00:00:00Z")
        self.assertGreater(result["age_days"], 365)
        self.assertEqual(result["nameservers"], ["ns1.example.com"])
        self.assertEqual(request.call_args.args[1], "https://rdap.org/domain/example.com")


if __name__ == "__main__":
    unittest.main()
