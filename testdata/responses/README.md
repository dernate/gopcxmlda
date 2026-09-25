# Response fixtures

Recorded-shape, **synthetic** SOAP responses used by the offline tests (`responses_test.go`).

They reproduce the exact structure of two real OPC XML-DA servers with different
SOAP stacks — namespace prefixes, `xsi:type` usage, attribute order, self-closing
elements, per-item `ResultID`s and faults — but all item names, values, handles,
vendor strings and host names are made up. No data from a real plant is stored here.

| Directory | Modeled on | Characteristics |
|---|---|---|
| `gsoap/` | gSOAP-style server | `SOAP-ENV:` prefix, `xsi:type` on every element, `Body id="_0"`, unprefixed `ResultID` |
| `axis/` | Apache Axis (Java) gateway | `soapenv:` prefix, no `xsi:type` outside `<Value>`, self-closing elements, `ResultID`s with locally declared prefixes, SOAP fault with HTTP 500 |

`gsoap/write.xml` is modeled on the specification's `WriteResponse` instead of a
recording: the offline and live suites never write to a real server.

To add a fixture for a new server, record the raw response of a **read-only**
operation, replace every real item name, value, handle and host name, and keep the
structure unchanged.
