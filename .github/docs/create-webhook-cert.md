# Create the CA Certificate for the Webhook Server

Request a new Certificate
```bash
openssl req -nodes -new -x509 -keyout ca.key -out ca.crt -subj "/CN=Admission Controller Webhook VaultRDB CA"
```
Transform the Certificate into a Base64 String.
```bash
echo "$(openssl base64 -A <"ca.crt")" > ca.crt.b64
```