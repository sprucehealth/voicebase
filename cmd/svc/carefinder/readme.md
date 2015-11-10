#### Overview

CareFinder is hosted under the same domain as the over arching website (https://www.sprucehealth.com) with the intention of bolstering the domain authority of the main website while also adding SEO optimized content for individual city pages.

It is run as a separate service, with all incoming requests being reverse proxied via the restapi service.

Content is created in a single database by manually curating the list of doctors, and then replicated to all environments.

#### Locally running CareFinder

1. Copy over the font files from the shared resource package (this is done at build time)
	```
		mkdir resources/static/fonts
		cp ../../../resources/static/fonts/* resources/static/fonts/
	```

2. Build the javascript and css files 
	```
		mkdir resources/static/js 
		./build_resources.sh
	```

3. Build the binary and run it

Include the following environment variables, and then run the built binary.

	```
	CAREFINDER_WEB_URL="http://127.0.0.1:8200/dermatologist-near-me"
	CAREFINDER_CONTENT_URL="https://d3l197bdp2dmpq.cloudfront.net"
	CAREFINDER_STATIC_RESOURCE_URL="http://127.0.0.1:8200/static"
	CAREFINDER_DB_HOST="provider-info.ckwporuc939i.us-east-1.rds.amazonaws.com"
	CAREFINDER_DB_NAME="npi"
	CAREFINDER_DB_USERNAME="spruce"

	For the following variables, get the value from someone on the backend team.
	CAREFINDER_DB_PASSWORD
	CAREFINDER_YELP_CONSUMER_KEY
	CAREFINDER_YELP_CONSUMER_SECRET
	CAREFINDER_YELP_TOKEN
	CAREFINDER_YELP_TOKEN_SECRET
	CAREFINDER_GOOGLE_STATIC_MAP_KEY
	CAREFINDER_GOOGLE_STATIC_MAP_URL_SIGNING_KEY
	```


#### Migrating contents from dev to staging/prod

1. Once you have updated the dev carefinder instance to your liking, take a compressed dump that can be imported staging/prod.

	```
	pg_dump -Fc -O -h provider-info.ckwporuc939i.us-east-1.rds.amazonaws.com -U spruce -d npi -t cities -t business_geocode -t namcs -t banner_image -t care_rating -t carefinder_doctor_info -t city_shortlist -t doctor_city_short_list -t doctor_short_list -t spruce_doctor_state_coverage -t spruce_review -t state -t top_skin_conditions_by_state   > npi.dump
	```

	Note that it is recommended to take the dump on the dev bastian box rather than on your local computer.

2. If you do end up taking the dump on your local computer, copy over the npi.dump onto the staging/prod bastian box.

3. Restore the staging/prod carefinder instance using the following command. Note that the `-c` option will first drop all tables and then recreate them from the dump. It is also assumed that you have installed the postgis extension at this point. If not, then run `create extension postgis` on the database before restoring from the dump.

	Restoring staging carefinder instance.
		```
		pg_restore --clean -h staging-carefinder-restored.ckwporuc939i.us-east-1.rds.amazonaws.com -U spruce -d npi npi.dump
		```

	Restoring prod carefinder instance:
		```
		pg_restore --clean -h prod-carefinder.ccvrwjdx3gvp.us-east-1.rds.amazonaws.com -U spruce -d npi npi.dump
		```

