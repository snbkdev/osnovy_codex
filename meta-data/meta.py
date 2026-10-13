#!/usr/bin/env python3
"""Print a web page's title and meta description using only the standard library."""

import argparse
import ssl
import sys
from html.parser import HTMLParser
from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.parse import urlsplit
from urllib.request import Request, urlopen


def create_ssl_context(cafile=None):
    context = ssl.create_default_context(cafile=cafile)
    if cafile is None:
        # Some Python installations lack a configured CA bundle.
        if sys.platform == "darwin" and Path("/etc/ssl/cert.pem").is_file():
            context.load_verify_locations(cafile="/etc/ssl/cert.pem")
        try:
            import certifi
        except ImportError:
            pass
        else:
            context.load_verify_locations(cafile=certifi.where())
    return context


class MetadataParser(HTMLParser):
    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.title_parts = []
        self.description = None
        self.in_title = False
        self.title_found = False

    def handle_starttag(self, tag, attrs):
        if tag == "title" and not self.title_found:
            self.in_title = True
            self.title_found = True
        if tag == "meta":
            attrs = dict(attrs)
            if (attrs.get("name") or "").strip().lower() == "description":
                if self.description is None:
                    self.description = attrs.get("content") or ""

    def handle_endtag(self, tag):
        if tag == "title":
            self.in_title = False

    def handle_data(self, data):
        if self.in_title:
            self.title_parts.append(data)


def main():
    parser = argparse.ArgumentParser(
        description="Вывести title и meta description HTML-страницы."
    )
    parser.add_argument("url", help="URL страницы (http:// или https://)")
    parser.add_argument("--ca-file", help="Файл доверенных CA-сертификатов в формате PEM")
    args = parser.parse_args()
    if urlsplit(args.url).scheme not in {"http", "https"} or not urlsplit(args.url).hostname:
        parser.error("Укажите полный URL с http:// или https://")

    request = Request(args.url, headers={"User-Agent": "PageMetadata/1.0"})
    try:
        context = create_ssl_context(args.ca_file)
        with urlopen(request, timeout=20, context=context) as response:
            content_type = response.headers.get_content_type()
            if content_type not in {"text/html", "application/xhtml+xml"}:
                raise ValueError(f"Ответ не является HTML: {content_type}")
            raw = response.read()
            encoding = response.headers.get_content_charset()
        if not encoding:
            # HTML may declare its encoding in a meta tag instead of HTTP headers.
            import re

            match = re.search(
                rb"<meta\b[^>]*\bcharset\s*=\s*[\"']?\s*([\w.-]+)",
                raw[:4096],
                re.IGNORECASE,
            )
            encoding = match.group(1).decode("ascii") if match else "utf-8-sig"
        metadata = MetadataParser()
        metadata.feed(raw.decode(encoding, errors="replace"))
    except (HTTPError, URLError, TimeoutError, OSError, ValueError, LookupError) as error:
        print(f"Ошибка: {error}", file=sys.stderr)
        reason = getattr(error, "reason", error)
        if isinstance(reason, ssl.SSLCertVerificationError):
            print(
                "Не удалось проверить сертификат. Установите доверенные CA: "
                "python3 -m pip install certifi\n"
                "Для собственного CA используйте --ca-file /path/to/ca.pem",
                file=sys.stderr,
            )
        return 1

    title = " ".join("".join(metadata.title_parts).split())
    description = " ".join((metadata.description or "").split())
    print(f"Title: {title or '(не найден)'}")
    print(f"Meta description: {description or '(не найден)'}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
