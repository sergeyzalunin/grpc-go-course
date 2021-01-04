#!/bin/bash
# ca.key: Certificate Authority private key file (this shouldn't be shared in real-life)
# ca.crt: Certificate Authority trust certificate (thisshould be shared with users in real-life)
# server.key: Server private key, password protected (this shouldn't be shared)
# server.csr: Server certificate signing request (this should be shared with the CA owner)
# server.crt: Server certificate signed by the CA (this would be sent back by the CA owner) - keep on server. 
# server.pem: Conversion of server.key into a format gRPC likes (this shouldn't be shared)

# Summary
# Private files: ca.key, server.key, server.pem, server.crt
# "Share" files: ca.crt, (needed by the client), server.csr (needed by the CA)
#CN_SERVER=localhost
#CA_PASSWORD=1234

CERT_PATH=ssl
#CA_KEY=$CERT_PATH/ca.key
#CA_CRT=$CERT_PATH/ca.crt
#SERVER_KEY=$CERT_PATH/server.key
#SERVER_CSR=$CERT_PATH/server.csr
#SERVER_CRT=$CERT_PATH/server.crt
#SERVER_PEM=$CERT_PATH/server.pem

rm -rf $CERT_PATH
mkdir -p $CERT_PATH
cd $CERT_PATH

# go get github.com/jsha/minica
minica --domains localhost

# Step 1: Generate the root certificate authority key with the set password + Trust Certificate (ca.crt)
#openssl genrsa -des3 -passout pass:$CA_PASSWORD -out $CA_KEY 4096
#openssl req -passin pass:$CA_PASSWORD -new -x509 -days 365 -key $CA_KEY -out $CA_CRT -subj "/CN=${CN_SERVER}"

# Step 2: Generate the server private key (server.key)
#openssl genrsa -passout pass:$CA_PASSWORD -des3 -out $SERVER_KEY 4096

# Step 3: Get a certificate signing request from the CA (server.csr)
#openssl req -passin pass:$CA_PASSWORD -new -key $SERVER_KEY -out $SERVER_CSR -subj "/CN=${CN_SERVER}"

# Step 4: Sign the certificate with the CA we created (it's called self signing) - server.crt
#openssl x509 -req -passin pass:$CA_PASSWORD -days 365 -in $SERVER_CSR -CA $CA_CRT -CAkey $CA_KEY -set_serial 01 -out $SERVER_CRT

# Step 5: Convert the server certificate to .pem format (server.pem) - usable by gRPC
#openssl pkcs8 -topk8 -nocrypt -passin pass:$CA_PASSWORD -in $SERVER_KEY -out $SERVER_PEM