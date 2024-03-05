package did

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"github.com/spf13/cobra"
)

func newVpCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-vp",
		Short: "generate vp [userDid] [usage] [challenge] [vc1 json] [vc2 json]...",
		Long:  "generate vp [userDid] [usage] [challenge] [vc1 json] [vc2 json]...",
		//Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			vcList := make([]string, len(args)-3)
			for i := 3; i < len(args); i++ {
				vcList[i-3] = args[i]
			}
			didDoc := generateVP(args[0], args[1], args[2], privateKey, vcList...)
			fmt.Println(didDoc)
			fmt.Println("-------------------")
			util.PrintPrettyJson(didDoc)
			return nil
		},
	}
	util.AttachFlags(cmd, flags, []string{flagPrivateKey, flagPubKey})
	return cmd
}
func generateVP(userDid string, usage string, challenge string, userPrvKeyPath string, vcJson ...string) string {
	vpTemp := `{
  "@context": [
    "https://www.w3.org/2018/credentials/v1",
    "https://www.w3.org/2018/credentials/examples/v1"
  ],
  "type": "VerifiablePresentation",
  "id": "https://example.com/presentations/123",
  "verifiableCredential": [
    %s
  ],
  "presentationUsage": "%s",
  "expirationDate": "2024-12-31T00:00:00Z",
  "verifier": "%s"
}`
	vcArray := strings.Join(vcJson, ",")
	vpJson := fmt.Sprintf(vpTemp, vcArray, usage, userDid)
	fmt.Println(vpJson)
	vp := NewVerifiablePresentation(vpJson)
	if vp == nil {
		panic("generate vp failed")
	}
	//签名
	signerPrvKey := getPrivateKey(userPrvKeyPath)
	signature := signVP(vp, signerPrvKey)
	vp.Proof = &Proof{
		Type:               "SM2Signature",
		Created:            getNowString(),
		ProofPurpose:       "authentication",
		VerificationMethod: userDid + "#keys-1",
		ProofValue:         signature,
		Challenge:          challenge,
	}
	signedVP, _ := json.Marshal(vp)
	return string(signedVP)
}

// VerifiablePresentation VP持有者展示的凭证
type VerifiablePresentation struct {
	rawData              json.RawMessage
	Context              []string               `json:"@context"`
	Type                 string                 `json:"type"`
	ID                   string                 `json:"id"`
	VerifiableCredential []VerifiableCredential `json:"verifiableCredential"`
	PresentationUsage    string                 `json:"presentationUsage,omitempty"`
	ExpirationDate       string                 `json:"expirationDate,omitempty"`
	Verifier             string                 `json:"verifier,omitempty"`
	Proof                *Proof                 `json:"proof,omitempty"`
}

// NewVerifiablePresentation 根据VP持有者展示的凭证json字符串创建VP持有者展示的凭证
func NewVerifiablePresentation(vpJson string) *VerifiablePresentation {
	var vp VerifiablePresentation
	err := json.Unmarshal([]byte(vpJson), &vp)
	if err != nil {
		return nil
	}
	vp.rawData = []byte(vpJson)
	return &vp
}
func signVP(vp *VerifiablePresentation, privKey crypto.PrivateKey) string {
	didDocumentWithoutProof := *vp
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
