Backend Monorepo
================

Setting up your environment & running the `gotour`
--------------------------------------------------

	$ brew update
	$ brew doctor # ensure there are no issues with Homebrew or your system
	$ brew install go mercurial
	# Add the following two lines to ~/.profile
	$ export GOPATH=$HOME/go
	$ go get golang.org/x/tour/gotour
	$ $HOME/go/bin/gotour # runs the gotour executable and opens it in a browser window


Updating and Creating Comment Diagrams
--------------------------------------

Sometimes a diagram is more useful than words to describe a bit of
functionality (for instance the flow of visit status). A great tool
to work with ascii diagrams is [Monodraw](http://monodraw.helftone.com/).
The existing diagrams in their original file format can be found on
Google Drive under Engineering/Backend/Diagrams.


Deploying
---------

Generally, continuous integration server handles deployment.

TODO: fill in this section with information about deployadmin cli


Static Resources on S3 and CloudFront
-------------------------------------

Static assets for all of our server environments (`dev`, `staging`, and `prod`) are hosted in a single S3 bucket called `spruce-static`.

You'll need an `Access Key ID` and `Secret` to view and make changes to that bucket. Once you have that, you can connect either via the `s3cmd` CLI tool (see `docker-ci/run.sh` for usage examples) or by an FTP client such as Transmit (create a new connection, choose the S3 protocol and `s3.amazonaws.com` as the server).

Here are two example URLs that point at the same file:

```
https://spruce-static.s3.amazonaws.com/curbside/devices.png
https://dlzz6qy5jmbag.cloudfront.net/curbside/devices.png
```

The latter is preferred, since it utilizes CloudFront.


Source Control and Code Review Workflow
---------------------------------------

## Source Control with Git

The `backend` monorepo follows a strategy such that features are squashed and merged to the `master` branch in a single commit.

## Code reviews

Code reviews are done through GitHub PRs. A PR needs approval and to have the latest changes from master before it can be merged.
