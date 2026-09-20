package rss

import "testing"

func TestParseRSS(t *testing.T) {
	items := Parse([]byte(`<?xml version="1.0"?><rss version="2.0"><channel>
		<item>
			<title>Hello &amp; Go</title>
			<link>https://example.com/a</link>
			<guid>a</guid>
			<description><![CDATA[<p>Body</p>]]></description>
			<enclosure url="https://example.com/a.jpg" type="image/jpeg"/>
		</item>
	</channel></rss>`))
	if len(items) != 1 || items[0].Title != "Hello & Go" || items[0].GUID != "a" || items[0].Body != "Body" || items[0].ImageURL == "" {
		t.Fatalf("%+v", items)
	}
}

func TestParseAtom(t *testing.T) {
	items := Parse([]byte(`<feed xmlns="http://www.w3.org/2005/Atom"><entry>
		<title>Atom</title><id>id-1</id>
		<link rel="alternate" href="https://example.com/b"/>
		<summary>Sum</summary>
	</entry></feed>`))
	if len(items) != 1 || items[0].GUID != "id-1" || items[0].Link != "https://example.com/b" || items[0].Body != "Sum" {
		t.Fatalf("%+v", items)
	}
}
