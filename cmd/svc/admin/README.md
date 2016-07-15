#Admin Service
The admin service is intended to expose an administrative interface for the Baymax system
## Getting Started
### Setting up LDAP
The admin service uses ldap for authentication.

Using docker-machine run and bootstrap LDAP with the following

```
docker kill ldap && \
docker rm ldap && \
docker run \
-p 389:389 \
--name ldap \
-e "SLAPD_PASSWORD=ldap" \
-e "SLAPD_DOMAIN=sprucehealth.com" \
-t dinkel/openldap
```

Once the server is running bootstrap the user data. This set defaults everything to be username and password `ldap`

```
echo "
# ORG UNIT ENTRY
dn: ou=People,dc=sprucehealth,dc=com
ou: People
objectClass: top
objectclass: organizationalUnit

# USER ENTRY
dn: uid=ldap,ou=People,dc=sprucehealth,dc=com
objectClass: person
objectClass: posixAccount
sn: ldap
cn: ldap
uid: ldap
uidNumber: 300
gidNumber: 300
homeDirectory: /home/ldap
userPassword: ldap
" | ldapadd -x -h $(docker-machine ip spruce) -D "cn=admin,dc=sprucehealth,dc=com" -w "ldap"
```

If LDAP is correctly configured the following query should succeed: `ldapsearch -x -D "cn=admin,dc=sprucehealth,dc=com" -w "ldap" -h $(docker-machine ip spruce) -b 'uid= ldap,ou=People,dc=sprucehealth,dc=com' -s base '(objectclass=*)'`

### Running the service
To run the service provide it the address of the LDAP server running inside docker

`go run main.go -env=dev -management_addr=:10000 -debug=true -ldap_addr=$(docker-machine ip spruce):389`