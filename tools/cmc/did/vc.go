package did

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"github.com/spf13/cobra"
)

func newVcCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-vc",
		Short: "generate vc [vcId] [vcType] [holderDid] [name] [id] [phoneNumber] [issuerDid] ",
		Long:  "generate vc [vcId] [vcType] [holderDid] [name] [id] [phoneNumber] [issuerDid]",
		Args:  cobra.ExactArgs(7),
		RunE: func(cmd *cobra.Command, args []string) error {
			didDoc := generateVC(args[0], args[1], args[2], args[3], args[4], args[5], args[6], privateKey)
			fmt.Println(didDoc)
			fmt.Println("-------------------")
			util.PrintPrettyJson(didDoc)
			return nil
		},
	}
	util.AttachFlags(cmd, flags, []string{flagPrivateKey, flagPubKey})
	return cmd
}

func generateVC(vcId string, vcType string, userDid, userName, id, phone, issuerDid string, issuerPrvKey string) string {
	vcTemp := `{
  "@context": [
    "https://www.w3.org/2018/credentials/v1",
    "https://www.w3.org/2018/credentials/examples/v1"
  ],
  "id": "%s",
  "type": ["VerifiableCredential", "IdentityCredential"],
  "issuer": "%s",
  "issuanceDate": "%s",
  "expirationDate": "2042-01-01T00:00:00Z",
  "credentialSubject": {
    "id": "%s",
    "name": "%s",
    "idNumber": "%s",
    "phoneNumber": "%s"
  },
  "template": {
    "id": "1",
    "name": "个人实名认证",
	"version": "1.0",
	"vcType":"%s"
  }
}`

	vcJson := fmt.Sprintf(vcTemp, vcId, issuerDid, getNowString(), userDid, userName, id, phone, vcType)
	vc := NewVerifiableCredential(vcJson)
	if vc == nil {
		panic("generate vc failed")
	}
	//签名
	signerPrvKey := getPrivateKey(issuerPrvKey)
	signature := signVC(vc, signerPrvKey)
	vc.Proof = &Proof{
		Type:               "SM2Signature",
		Created:            getNowString(),
		ProofPurpose:       "assertionMethod",
		VerificationMethod: issuerDid + "#keys-1",
		ProofValue:         signature,
	}
	signedVC, _ := json.Marshal(vc)
	return string(signedVC)
}
func signVC(vc *VerifiableCredential, privKey crypto.PrivateKey) string {
	didDocumentWithoutProof := *vc
	didDocumentWithoutProof.Proof = nil
	didDocBytes, err := json.Marshal(didDocumentWithoutProof)
	if err != nil {
		panic(err)
	}
	sig, err := privKey.Sign(didDocBytes)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(sig)
}

// VerifiableCredential VC凭证，证书
type VerifiableCredential struct {
	rawData           json.RawMessage
	Context           []string               `json:"@context"`
	ID                string                 `json:"id"`
	Type              []string               `json:"type"`
	Issuer            string                 `json:"issuer"`
	IssuanceDate      string                 `json:"issuanceDate"`
	ExpirationDate    string                 `json:"expirationDate"`
	CredentialSubject map[string]interface{} `json:"credentialSubject"`
	Template          *struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Version string `json:"version"`
		VcType  string `json:"vcType"`
	} `json:"template,omitempty"`
	Proof *Proof `json:"proof,omitempty"`
}

// NewVerifiableCredential 根据VC凭证json字符串创建VC凭证
func NewVerifiableCredential(vcJson string) *VerifiableCredential {
	var vc VerifiableCredential
	err := json.Unmarshal([]byte(vcJson), &vc)
	if err != nil {
		return nil
	}
	vc.rawData = []byte(vcJson)
	return &vc
}

// GetCredentialSubjectID 获取VC凭证的持有者DID
func (vc *VerifiableCredential) GetCredentialSubjectID() string {
	return vc.CredentialSubject["id"].(string)
}
