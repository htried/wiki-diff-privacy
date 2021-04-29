# wiki-diff-privacy
Toolforge interface / API for testing out how differential privacy might be applied to Wikimedia data. 

## How to use
To run this program locally, you need to have golang, mysql, and all requisite packages installed. You will also need a file called `replica.my.cnf` containing the following fields:

user = your mysql username

password = your mysql password

Then, navigate to the base directory (`cd /path/to/wiki-diff-privacy`) and:
1) `go run init_db.go` — initializes synthetic data db (privacy unit is either a pageview) and output db
2) `go run beam.go` — does normal and differentially private counts of the existing dbs. Warning: this is an expensive computation — on a newish macbook, a language with around 500,000 views takes >90 seconds to count. To do all of the languages I've laid out here, it usually takes between 15-20 minutes.
3) `go run clean_db.go` — removes old dbs and synthetic data, which can take up a lot of space
4) `go run server.go` — runs the server. Should be accessible at 127.0.0.1:8000 in a browser.

Information for how to deploy this site on Cloud VPS is available at `config/README.md`.

## License
The source code for this interface is released under the [MIT license](https://github.com/geohci/wiki-diff-privacy/blob/main/LICENSE).

Screenshots of the results in the API may be used without attribution, but a link back to the application would be appreciated.