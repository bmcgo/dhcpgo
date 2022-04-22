SUBJ=/C=US/ST=SomeState/L=SomeLocation/O=dhcpg/OU=dhcp

all:
	@echo usage

dev/ca/ca-key.pem:
	mkdir -p dev/ca
	echo "subjectAltName = IP:127.0.0.1" > dev/ca/file.ext
	openssl ecparam -name secp521r1 -genkey -noout -out dev/ca/ca-key.pem

dev/ca/ca.pem: dev/ca/ca-key.pem
	openssl req -new -x509 -nodes -key dev/ca/ca-key.pem -out dev/ca/ca.pem -subj "${SUBJ}" -addext "subjectAltName = IP:127.0.0.1"

dev/ca/etcd-key.pem: dev/ca/ca.pem
	openssl ecparam -name secp521r1 -genkey -noout -out dev/ca/etcd-key.pem

dev/ca/client-key.pem: dev/ca/ca.pem
	openssl ecparam -name secp521r1 -genkey -noout -out dev/ca/client-key.pem

dev/ca/etcd-cert.pem: dev/ca/etcd-key.pem
	openssl req -new -key dev/ca/etcd-key.pem -nodes -out dev/ca/etcd-csr.pem -subj "${SUBJ}" -addext "subjectAltName = IP:127.0.0.1"
	openssl x509 -req -in dev/ca/etcd-csr.pem -days 3650 -CA dev/ca/ca.pem -CAkey dev/ca/ca-key.pem -set_serial 42 -out dev/ca/etcd-cert.pem -extfile dev/ca/file.ext
	rm dev/ca/etcd-csr.pem


dev/ca/client-cert.pem: dev/ca/client-key.pem
	openssl req -new -key dev/ca/client-key.pem -nodes -out dev/ca/client-csr.pem -subj "${SUBJ}" -addext "subjectAltName = IP:127.0.0.1"
	openssl x509 -req -in dev/ca/client-csr.pem -days 3650 -CA dev/ca/ca.pem -CAkey dev/ca/ca-key.pem -set_serial 43 -out dev/ca/client-cert.pem -extfile dev/ca/file.ext
	rm dev/ca/client-csr.pem

etcd-cert: dev/ca/etcd-cert.pem dev/ca/client-cert.pem

run-etcd: etcd-cert
	etcd 	--data-dir dev/etcd-data \
		--listen-client-urls https://127.0.0.1:2379 \
		--advertise-client-urls https://127.0.0.1:2379 \
		--trusted-ca-file dev/ca/ca.pem \
		--cert-file dev/ca/etcd-cert.pem \
		--client-cert-auth \
		--key-file dev/ca/etcd-key.pem

run-dhcpg:
	sudo ./dhcpg \
		--etcd-ca dev/ca/ca.pem \
		--etcd-key dev/ca/client-key.pem \
		--etcd-cert dev/ca/client-cert.pem \
		--etcd-url https://127.0.0.1:2379
