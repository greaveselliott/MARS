# Embedded Dashboard Browser Assets

These files are embedded into the MARS binary so the dashboard never needs an
outbound CDN request. Hashes are SHA-256 over the committed bytes.

| File | Version | Immutable source | SHA-256 | License |
| --- | --- | --- | --- | --- |
| `htmx-2.0.4.min.js` | htmx 2.0.4 | `https://raw.githubusercontent.com/bigskysoftware/htmx/v2.0.4/dist/htmx.min.js` | `e209dda5c8235479f3166defc7750e1dbcd5a5c1808b7792fc2e6733768fb447` | Zero-Clause BSD, `htmx-LICENSE.txt` |
| `chart-4.4.7.umd.js` | Chart.js 4.4.7 | `https://registry.npmjs.org/chart.js/-/chart.js-4.4.7.tgz`, member `package/dist/chart.umd.js` | `2812cb8825fdc57469eb2f7bb055e9429244e599920511ee477e828499b632cb` | MIT, `chartjs-LICENSE.md` |

The downloaded Chart.js package archive was independently checked as
`af01b7afdbb70017c079b0dc78ef883f860d28fa2644ed78906a39090d07134f`.
This inventory records upstream provenance only; it does not make or change
ownership-dependent project copyright claims.
