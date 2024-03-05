## DID文档
### 通过公私钥文件生成DID Document
```bash
./cmc did pk-to-did --pub-key=./admin1/admin1.pem --prv-key=./admin1/admin1.key
```

## VC
### 通过传入vcid,持有人DID，实名认证姓名、身份证号，电话号码，颁发机构DID 生成VC
```bash
./cmc did generate-vc "vcId123" "did:cnbn:5eb4e668952dcef3018a5bc03ca9517eff1cbfa7" "Devin" "511112188501010001" "13811888888" "did:cnbn:eadf82170c8d6f2ea9349f921be50967ba62b18a" --prv-key=./admin3/admin3.key
```
其中的admin3是作为Issuer，颁发者的私钥


##VP
生成了VC之后，我们可以通过传入:验证者DID，用途，随机数挑战，VC的完整内容，生成VP
```bash
./cmc did generate-vp "did:cnbn:5eb4e668952dcef3018a5bc03ca9517eff1cbfa7" "租房" "123" "{\"@context\":[\"https://www.w3.org/2018/credentials/v1\",\"https://www.w3.org/2018/credentials/examples/v1\"],\"id\":\"https://example.com/credentials/123\",\"type\":[\"VerifiableCredential\",\"IdentityCredential\"],\"issuer\":\"did:cnbn:eadf82170c8d6f2ea9349f921be50967ba62b18a\",\"issuanceDate\":\"2023-01-01T00:00:00Z\",\"expirationDate\":\"2042-01-01T00:00:00Z\",\"credentialSubject\":{\"id\":\"did:cnbn:5eb4e668952dcef3018a5bc03ca9517eff1cbfa7\",\"idNumber\":\"511112188501010001\",\"name\":\"Devin\",\"phoneNumber\":\"13811888888\"},\"template\":{\"id\":\"1\",\"name\":\"个人实名认证\",\"version\":\"1.0\"},\"proof\":{\"type\":\"SM2Signature\",\"created\":\"2023-01-01T00:00:00Z\",\"proofPurpose\":\"assertionMethod\",\"verificationMethod\":\"did:cnbn:eadf82170c8d6f2ea9349f921be50967ba62b18a#keys-1\",\"proofValue\":\"MEUCIQCNZ7sSa4vcC03HYVMQdN/B3t1e25fnB3H6L77s3eGUZgIgHhFn84qtg/meCNNjDQKz+X/WUWKJSBmNK/b4ZIlytnM=\"}}" --prv-key=./admin2/admin2.key
```
其中的admin2是作为Holder持有者的私钥

如果要生成多个VC的VP，可以通过传入多个VC的完整内容，用空格分隔
```bash
./cmc did generate-vp "did:cnbn:5eb4e668952dcef3018a5bc03ca9517eff1cbfa7" "租房" "123" "{\"@context\":[\"https://www.w3.org/2018/credentials/v1\",\"https://www.w3.org/2018/credentials/examples/v1\"],\"id\":\"https://example.com/credentials/123\",\"type\":[\"VerifiableCredential\",\"IdentityCredential\"],\"issuer\":\"did:cnbn:eadf82170c8d6f2ea9349f921be50967ba62b18a\",\"issuanceDate\":\"2023-01-01T00:00:00Z\",\"expirationDate\":\"2042-01-01T00:00:00Z\",\"credentialSubject\":{\"id\":\"did:cnbn:5eb4e668952dcef3018a5bc03ca9517eff1cbfa7\",\"idNumber\":\"511112188501010001\",\"name\":\"Devin\",\"phoneNumber\":\"13811888888\"},\"template\":{\"id\":\"1\",\"name\":\"个人实名认证\",\"version\":\"1.0\"},\"proof\":{\"type\":\"SM2Signature\",\"created\":\"2023-01-01T00:00:00Z\",\"proofPurpose\":\"assertionMethod\",\"verificationMethod\":\"did:cnbn:eadf82170c8d6f2ea9349f921be50967ba62b18a#keys-1\",\"proofValue\":\"MEUCIQCNZ7sSa4vcC03HYVMQdN/B3t1e25fnB3H6L77s3eGUZgIgHhFn84qtg/meCNNjDQKz+X/WUWKJSBmNK/b4ZIlytnM=\"}}" "VC2内容" "VC3内容" --prv-key=./admin2/admin2.key
```
