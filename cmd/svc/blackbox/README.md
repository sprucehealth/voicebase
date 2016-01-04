#Black Box Testing Service
## Local Development
### Database Setup
If you don't already have it setup install `flyway`

```
$ brew update
$ brew install flyway
```

In `mysql` execute the following from the `root account`.

```
CREATE SCHEMA blackbox;
CREATE USER 'blackbox'@'localhost' IDENTIFIED BY 'blackbox';
GRANT ALL PRIVILEGES ON blackbox.* TO 'blackbox'@'localhost';
```

### Initialize the schema

```
$ flyway -url=jdbc:mysql://localhost:3306/blackbox -user=blackbox -password=blackbox -locations=filesystem:$GOPATH/src/github.com/sprucehealth/backend/cmd/svc/blackbox/internal/dal/postgres -validateOnMigrate=true migrate
```

## Testing Strategies
### Random Valid Request Generation

It is useful to establish functions that generate random valid requests (not fuzz). Requests with optional fields should randomly add and omit those fields.

```
func optionalTokenAttributes() map[string]string {
	// token attributes is optional
	var tokenAttributes map[string]string
	if harness.RandBool() {
		tokenAttributes = make(map[string]string)
		for i := int64(0); i < harness.RandInt64N(maxAuthTokenAttributes); i++ {
			tokenAttributes[harness.RandLengthString(maxAuthTokenAttributeKeyLength)] = harness.RandLengthString(maxAuthTokenAttributeValueLength)
		}
	}
	return tokenAttributes
}

func randomValidCreateAccountRequest() *auth.CreateAccountRequest {
	return &auth.CreateAccountRequest{
		FirstName:       harness.RandLengthString(maxAccountFirstNameSize),
		LastName:        harness.RandLengthString(maxAccountLastNameSize),
		Email:           harness.RandEmail(),
		PhoneNumber:     harness.RandPhoneNumber(),
		Password:        harness.RandLengthString(maxAccountPasswordLength),
		TokenAttributes: optionalTokenAttributes(),
	}
}
``` 