# Embedded Dashboard Browser Assets

These files are embedded into the MARS binary so the dashboard never needs an
outbound CDN request. Hashes are SHA-256 over the committed bytes.

| File | Version | Source commit | Upstream package | Committed SHA-256 | License |
| --- | --- | --- | --- | --- | --- |
| `htmx-2.0.4.min.js` | htmx 2.0.4 | `b82cf843e47e575dd8c2ad8fee547d8e2c3bb87f` | `htmx.org-2.0.4.tgz`, member `package/dist/htmx.min.js` | `e209dda5c8235479f3166defc7750e1dbcd5a5c1808b7792fc2e6733768fb447` | Zero-Clause BSD, `htmx-LICENSE.txt`, SHA-256 `d3d2456f76414f2456104660ebd65aff1c04cd7966b942bdabd63f3cdb316a38` |
| `chart-4.4.7.umd.js` | Chart.js 4.4.7 | `57b5c5b78fb2d8504f556bef6e4177735d9929ea` | `chart.js-4.4.7.tgz`, member `package/dist/chart.umd.js` | `2812cb8825fdc57469eb2f7bb055e9429244e599920511ee477e828499b632cb` | MIT, `chartjs-LICENSE.md`, SHA-256 `41a84aa2caba645f966a18d9c2056b73e6d3a81d80bc0046bc0011a2634d4cce` |

The npm package records and independently checked archive SHA-256 values are:

- htmx: integrity `sha512-HLxMCdfXDOJirs3vBZl/ZLoY+c7PfM4Ahr2Ad4YXh6d22T5ltbTXFFkpx9Tgb2vvmWFMbIc3LqN2ToNkZJvyYQ==`; archive SHA-256 `a34988d1b9f005a458d593aa8d7486537cb878b8ac3b82703db1268983123ecc`.
- Chart.js: integrity `sha512-pwkcKfdzTMAU/+jNosKhNL2bHtJc/sSmYgVbuGTEDhzkrhmyihmP7vUc/5ZK9WopidMDHNe3Wm7jOd/WhuHWuw==`; archive SHA-256 `af01b7afdbb70017c079b0dc78ef883f860d28fa2644ed78906a39090d07134f`.

This inventory records upstream provenance only; it does not make or change
ownership-dependent project copyright claims.
