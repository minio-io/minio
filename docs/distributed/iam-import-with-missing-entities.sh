#!/bin/bash

if [ -n "$TEST_DEBUG" ]; then
	set -x
fi

pkill minio
docker rm -f $(docker ps -aq)
rm -rf /tmp/ldap{1..4}
rm -rf /tmp/ldap1{1..4}

if [ ! -f ./mc ]; then
	wget --quiet -O mc https://dl.minio.io/client/mc/release/linux-amd64/mc &&
		chmod +x mc
fi

mc -v

# Start LDAP server
echo "Copying docs/distributed/samples/bootstrap-complete.ldif => minio-iam-testing/ldap/50-bootstrap.ldif"
cp docs/distributed/samples/bootstrap-complete.ldif minio-iam-testing/ldap/50-bootstrap.ldif || exit 1
cd ./minio-iam-testing
make docker-images
make docker-run
cd -

# Start MinIO instance
export CI=true
(minio server --address :22000 --console-address :10000 http://localhost:22000/tmp/ldap{1...4} 2>&1 >/dev/null) &
sleep 30
./mc ready myminio
./mc alias set myminio http://localhost:22000 minioadmin minioadmin

./mc idp ldap add myminio server_addr=localhost:1389 server_insecure=on lookup_bind_dn=cn=admin,dc=min,dc=io lookup_bind_password=admin user_dn_search_base_dn=dc=min,dc=io user_dn_search_filter="(uid=%s)" group_search_base_dn=ou=swengg,dc=min,dc=io group_search_filter="(&(objectclass=groupOfNames)(member=%d))"
./mc admin service restart myminio --quiet --disable-pager
./mc ready myminio
./mc admin cluster iam import myminio docs/distributed/samples/myminio-iam-info.zip
sleep 10

# Verify the list of users and service accounts from the import
./mc admin user list myminio
USER_COUNT=$(./mc admin user list myminio | wc -l)
if [ "${USER_COUNT}" -ne 2 ]; then
	echo "BUG: Expected no of users: 2 Found: ${USER_COUNT}"
	exit 1
fi
./mc admin user svcacct list myminio "uid=bobfisher,ou=people,ou=hwengg,dc=min,dc=io" --json
SVCACCT_COUNT_1=$(./mc admin user svcacct list myminio "uid=bobfisher,ou=people,ou=hwengg,dc=min,dc=io" --json | jq '.accessKey' | wc -l)
if [ "${SVCACCT_COUNT_1}" -ne 2 ]; then
	echo "BUG: Expected svcacct count for 'uid=bobfisher,ou=people,ou=hwengg,dc=min,dc=io': 2. Found: ${SVCACCT_COUNT_1}"
	exit 1
fi
./mc admin user svcacct list myminio "uid=dillon,ou=people,ou=swengg,dc=min,dc=io" --json
SVCACCT_COUNT_2=$(./mc admin user svcacct list myminio "uid=dillon,ou=people,ou=swengg,dc=min,dc=io" --json | jq '.accessKey' | wc -l)
if [ "${SVCACCT_COUNT_2}" -ne 2 ]; then
	echo "BUG: Expected svcacct count for 'uid=dillon,ou=people,ou=swengg,dc=min,dc=io': 2. Found: ${SVCACCT_COUNT_2}"
	exit 1
fi

# Kill MinIO and LDAP to start afresh with missing groups/DN
pkill minio
docker rm -f $(docker ps -aq)
rm -rf /tmp/ldap{1..4}

# Deploy the LDAP config witg missing groups/DN
echo "Copying docs/distributed/samples/bootstrap-partial.ldif => minio-iam-testing/ldap/50-bootstrap.ldif"
cp docs/distributed/samples/bootstrap-partial.ldif minio-iam-testing/ldap/50-bootstrap.ldif || exit 1
cd ./minio-iam-testing
make docker-images
make docker-run
cd -

(minio server --address ":24000" --console-address :10000 http://localhost:24000/tmp/ldap1{1...4} 2>&1 >/dev/null) &
sleep 30
./mc ready myminio1
./mc alias set myminio1 http://localhost:24000 minioadmin minioadmin

./mc idp ldap add myminio1 server_addr=localhost:1389 server_insecure=on lookup_bind_dn=cn=admin,dc=min,dc=io lookup_bind_password=admin user_dn_search_base_dn=dc=min,dc=io user_dn_search_filter="(uid=%s)" group_search_base_dn=ou=hwengg,dc=min,dc=io group_search_filter="(&(objectclass=groupOfNames)(member=%d))"
./mc admin service restart myminio1 --quiet --disable-pager
./mc ready myminio1
./mc admin cluster iam import myminio1 docs/distributed/samples/myminio-iam-info.zip
sleep 10

# Verify the list of users and service accounts from the import
./mc admin user list myminio1
USER_COUNT=$(./mc admin user list myminio1 | wc -l)
if [ "${USER_COUNT}" -ne 1 ]; then
	echo "BUG: Expected no of users: 1 Found: ${USER_COUNT}"
	exit 1
fi
./mc admin user svcacct list myminio1 "uid=bobfisher,ou=people,ou=hwengg,dc=min,dc=io" --json
SVCACCT_COUNT_1=$(./mc admin user svcacct list myminio1 "uid=bobfisher,ou=people,ou=hwengg,dc=min,dc=io" --json | jq '.accessKey' | wc -l)
if [ "${SVCACCT_COUNT_1}" -ne 2 ]; then
	echo "BUG: Expected svcacct count for 'uid=bobfisher,ou=people,ou=hwengg,dc=min,dc=io': 2. Found: ${SVCACCT_COUNT_1}"
	exit 1
fi
./mc admin user svcacct list myminio1 "uid=dillon,ou=people,ou=swengg,dc=min,dc=io" --json
SVCACCT_COUNT_2=$(./mc admin user svcacct list myminio1 "uid=dillon,ou=people,ou=swengg,dc=min,dc=io" --json | jq '.accessKey' | wc -l)
if [ "${SVCACCT_COUNT_2}" -ne 0 ]; then
	echo "BUG: Expected svcacct count for 'uid=dillon,ou=people,ou=swengg,dc=min,dc=io': 0. Found: ${SVCACCT_COUNT_2}"
	exit 1
fi

# Finally kill running processes
pkill minio
docker rm -f $(docker ps -aq)
