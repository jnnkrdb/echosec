# Operator-SDK

## Install Operator-SDK

[See here](https://sdk.operatorframework.io/docs/installation/)
```bash
export ARCH=$(case $(uname -m) in x86_64) echo -n amd64 ;; aarch64) echo -n arm64 ;; *) echo -n $(uname -m) ;; esac)
export OS=$(uname | awk '{print tolower($0)}')
export OPERATOR_SDK_DL_URL=https://github.com/operator-framework/operator-sdk/releases/download/v1.32.0
curl -LO ${OPERATOR_SDK_DL_URL}/operator-sdk_${OS}_${ARCH}
gpg --keyserver keyserver.ubuntu.com --recv-keys 052996E2A20B5C7E
curl -LO ${OPERATOR_SDK_DL_URL}/checksums.txt
curl -LO ${OPERATOR_SDK_DL_URL}/checksums.txt.asc
gpg -u "Operator SDK (release) <cncf-operator-sdk@cncf.io>" --verify checksums.txt.asc
grep operator-sdk_${OS}_${ARCH} checksums.txt | sha256sum -c -
chmod +x operator-sdk_${OS}_${ARCH} && sudo mv operator-sdk_${OS}_${ARCH} /usr/local/bin/operator-sdk
operatos-sdk version
```

## Using the Commandline for operator-sdk

```bash
operator-sdk init --domain jnnkrdb.de --repo github.com/jnnkrdb/echosec
```

```bash
operator-sdk create api --group="core" --version v1 --kind Namespace --resource=false --controller=true
operator-sdk create webhook --group core --version v1 --kind Namespace --defaulting --programmatic-validation
```
