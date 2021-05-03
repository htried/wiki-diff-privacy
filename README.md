# wiki-diff-privacy
Toolforge interface / API for testing out how differential privacy might be applied to Wikimedia data. 

## How to run locally 
To run this program locally, you need to have golang, mysql, and all requisite packages installed. You will also need a file called `replica.my.cnf` containing the following fields:

user = your mysql username

password = your mysql password

Then, navigate to the base directory (`cd /path/to/wiki-diff-privacy`) and:
1) `go run init_db.go` — initializes synthetic data db (privacy unit is either a pageview) and output db
2) `go run beam.go` — does normal and differentially private counts of the existing dbs. Warning: this is an expensive computation — on a newish macbook, a language with around 500,000 views takes >90 seconds to count. To do all of the languages I've laid out here, it usually takes between 15-20 minutes.
3) `go run clean_db.go` — removes old dbs and synthetic data, which can take up a lot of space
4) `go run server.go` — runs the server. Should be accessible at 127.0.0.1:8000 in a browser.

## How to get this project set up on Cloud VPS:

On your local machine, wherever you keep your code:
```
git clone https://github.com/htried/wiki-diff-privacy.git
scp wiki-diff-privacy/config/*.sh <username>@diff-privacy-beam-test.wmf-research-tools.eqiad1.wikimedia.cloud:/home/<username>/

# these two are secret files, contact htriedman-ctr@wikimedia.org to access them
scp wiki-diff-privacy/config/replica.my.cnf <username>@diff-privacy-beam-test.wmf-research-tools.eqiad1.wikimedia.cloud:/home/<username>/
scp wiki-diff-privacy/config/config.sql <username>@diff-privacy-beam-test.wmf-research-tools.eqiad1.wikimedia.cloud:/home/<username>/
```

Now ssh into your Cloud VPS machine:
```
ssh <username>@diff-privacy-beam-test.wmf-research-tools.eqiad1.wikimedia.cloud
sudo bash cloudvps_setup.sh
sudo bash update_data.sh
```

At this point, the website should be working fine. You can check that by navigating to `diff-privacy-beam.wmcloud.org` on your browser.

Finally, set up a cron job to update the data every day at UTC+0900
```
sudo crontab -e
add a line reading “0 9 * * * /home/<username>/update_data.sh” to the end of the crontab
```

If you make changes to the codebase on your local machine, you can see those changes reflected in production by ssh-ing into your Cloud VPS machine and running `sudo bash release.sh`, which will pull from github and restart the nginx server with the relevant changes.

## License
The source code for this interface is released under the [MIT license](https://github.com/geohci/wiki-diff-privacy/blob/main/LICENSE).

Screenshots of the results in the API may be used without attribution, but a link back to the application would be appreciated.