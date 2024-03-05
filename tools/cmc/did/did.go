package did

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/evmutils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	sdkConfPath  string
	pubKey       string
	privateKey   string
	contractName string
	serviceUrl   string
)

const (
	flagSdkConfPath  = "sdk-conf-path"
	flagPubKey       = "pub-key"
	flagPrivateKey   = "prv-key"
	flagContractName = "contract-name"
	flagServiceUrl   = "service-url"
)

var flags *pflag.FlagSet

func init() {
	flags = &pflag.FlagSet{}

	flags.StringVar(&pubKey, flagPubKey, "", "public key files, separated by comma")
	flags.StringVar(&privateKey, flagPrivateKey, "", "private key files, separated by comma")
	flags.StringVar(&sdkConfPath, flagSdkConfPath, "", "specify sdk config path")
	flags.StringVar(&contractName, flagContractName, "", "contract name")
	flags.StringVar(&serviceUrl, flagServiceUrl, "", "issuer service url")

	if sdkConfPath == "" {
		sdkConfPath = util.EnvSdkConfPath
	}
}

//cmc did generate --pub-keys=pub1.pem,pub2.pem --prv-keys=prv1.pem,prv2.pem
//cmc did add --contract-name=DID --sdk-conf-path=../conf/sdk.yaml
//cmc did update
//cmc did vc generate
//cmc did vp generate

// NewDIdCMD new address parse command
func NewDIdCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "did",
		Short: "did generate command",
		Long:  "did generate command",
	}

	cmd.AddCommand(newDidCMD())
	cmd.AddCommand(newVcCMD())
	cmd.AddCommand(newVpCMD())
	//cmd.AddCommand(newCert2AddrCMD())
	//cmd.AddCommand(newName2AddrCMD())

	return cmd
}

//func newDidCMD() *cobra.Command {
//	cmd := &cobra.Command{
//		Use:   "pk-to-did",
//		Short: "get did document from public key file or pem string",
//		Long:  "get did document from public key file or pem string",
//		Args:  cobra.ExactArgs(0),
//		RunE: func(cmd *cobra.Command, args []string) error {
//			didDoc := generateDidDocument(pubKey, privateKey)
//			fmt.Println(didDoc)
//			fmt.Println("-------------------")
//			util.PrintPrettyJson(didDoc)
//			return nil
//		},
//	}
//	util.AttachFlags(cmd, flags, []string{flagPrivateKey, flagPubKey, flagServiceUrl})
//	return cmd
//}

func newDidCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pk-to-did",
		Short: "从公钥文件或 PEM 字符串获取 DID 文档",
		Long:  "从公钥文件或 PEM 字符串获取 DID 文档",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			pubKeyFiles, err := cmd.Flags().GetString("pub-key")
			if err != nil {
				return err
			}
			prvKeyFiles, err := cmd.Flags().GetString("prv-key")
			if err != nil {
				return err
			}

			pubKeys := strings.Split(pubKeyFiles, ",")
			prvKeys := strings.Split(prvKeyFiles, ",")

			if len(pubKeys) != len(prvKeys) {
				return errors.New("the number of public and private keys does not match")
			}

			didDoc := generateDidDocument(pubKeys, prvKeys)
			fmt.Println(didDoc)
			fmt.Println("-------------------")
			util.PrintPrettyJson(didDoc)
			return nil
		},
	}
	util.AttachFlags(cmd, flags, []string{flagPrivateKey, flagPubKey, flagServiceUrl})
	return cmd
}

// GenerateDidDocument 创建当前用户的DID文档
func generateDidDocument(pubKeyPaths, prvKeyPaths []string) string {
	didDocTemp := `{
  "@context": "https://www.w3.org/ns/did/v1",
  "id": "%s",
  "verificationMethod": [%s],
  "authentication": [%s],
  "controller": [%s]
}`
	var verificationMethods, authentications, controllers []string

	//本DID
	ownerPubKeyPath := pubKeyPaths[0]
	ownerPrvKeyPath := prvKeyPaths[0]
	ownerPubKey := getPubKey(ownerPubKeyPath)
	ownerAddr := getAddress(ownerPubKey)
	ownerDid := "did:cnbn:" + ownerAddr

	for i := 0; i < len(pubKeyPaths); i++ {
		pubKeyPath := pubKeyPaths[i]
		pubKeyPem := string(getPubKeyPem(pubKeyPath))
		userPubKey := getPubKey(pubKeyPath)
		userAddr := getAddress(userPubKey)
		userDid := "did:cnbn:" + userAddr

		pubKeyPem = strings.ReplaceAll(strings.ReplaceAll(pubKeyPem, "\n", "\\n"), "\r", "")

		verificationMethod := fmt.Sprintf(`{
      "id": "%s#keys-%d",
      "type": "SM2VerificationKey2020",
      "controller": "%s",
      "publicKeyPem": "%s",
      "address": "%s"
    }`, ownerDid, i+1, userDid, pubKeyPem, userAddr)

		authentication := fmt.Sprintf(`"%s#keys-%d"`, ownerDid, i+1)

		verificationMethods = append(verificationMethods, verificationMethod)
		authentications = append(authentications, authentication)
	}

	controller := fmt.Sprintf(`"%s"`, ownerDid)
	controllers = append(controllers, controller)

	verificationMethodsStr := strings.Join(verificationMethods, ",\n")
	authenticationsStr := strings.Join(authentications, ",\n")
	controllersStr := strings.Join(controllers, ",\n")

	didDocJson := fmt.Sprintf(didDocTemp, ownerDid, verificationMethodsStr, authenticationsStr, controllersStr)

	didDoc := NewDIDDocument(didDocJson)
	if didDoc == nil {
		panic("generate did document failed")
	}
	if len(serviceUrl) > 0 {
		didDoc.Service = []struct {
			ID              string `json:"id"`
			Type            string `json:"type"`
			ServiceEndpoint string `json:"serviceEndpoint"`
		}{
			{
				ID:              serviceUrl,
				Type:            "IssuerService",
				ServiceEndpoint: serviceUrl,
			},
		}
	}

	//for i := 0; i < len(pubKeyPaths); i++ {
	//	pubKeyPath := pubKeyPaths[i]
	//	prvKeyPath := prvKeyPaths[i]
	//
	//	signerPk := getPubKey(pubKeyPath)
	//	signerAddr := getAddress(signerPk)
	//	//signerDid := "did:cnbn:" + signerAddr
	//	signature := signDidDocument(didDoc, getPrivateKey(prvKeyPath))
	//	proof := &Proof{
	//		Type:               "SM2Signature",
	//		Created:            getNowString(),
	//		ProofPurpose:       "verificationMethod",
	//		VerificationMethod: signerDid + "#keys-1",
	//		ProofValue:         signature,
	//	}
	//
	//	proofj, _ := json.Marshal(proof)
	//	didDoc.Proof = proofj
	//}

	// 按照主Did的私钥来签名
	signature := signDidDocument(didDoc, getPrivateKey(ownerPrvKeyPath))
	proof := &Proof{
		Type:               "SM2Signature",
		Created:            getNowString(),
		ProofPurpose:       "verificationMethod",
		VerificationMethod: ownerDid + "#keys-1",
		ProofValue:         signature,
	}

	proofj, _ := json.Marshal(proof)
	didDoc.Proof = proofj

	newDidDocument, _ := json.Marshal(didDoc)
	return string(newDidDocument)
}

//// GenerateDidDocument 创建当前用户的DID文档
//func generateDidDocument(pubKeyPath, prvKeyPath string) string {
//	didDocTemp := `{
//  "@context": "https://www.w3.org/ns/did/v1",
//  "id": "%s",
//  "verificationMethod": [
//    {
//      "id": "%s#keys-1",
//      "type": "SM2VerificationKey2020",
//      "controller": "%s",
//      "publicKeyPem": "%s",
//      "address": "%s"
//    }
//  ],
//  "authentication": [
//    "%s#keys-1"
//  ],
//  "controller":["%s"]
//}`
//	pubKeyPem := string(getPubKeyPem(pubKeyPath))
//	userPubKey := getPubKey(pubKeyPath)
//	userAddr := getAddress(userPubKey)
//	userDid := "did:cnbn:" + userAddr
//
//	pubKeyPem = strings.ReplaceAll(strings.ReplaceAll(pubKeyPem, "\n", "\\n"), "\r", "")
//	didDocJson := fmt.Sprintf(didDocTemp, userDid, userDid, userDid, pubKeyPem, userAddr, userDid, userDid)
//	didDoc := NewDIDDocument(didDocJson)
//	if didDoc == nil {
//		panic("generate did document failed")
//	}
//	if len(serviceUrl) > 0 {
//		didDoc.Service = []struct {
//			ID              string `json:"id"`
//			Type            string `json:"type"`
//			ServiceEndpoint string `json:"serviceEndpoint"`
//		}{
//			{
//				ID:              serviceUrl,
//				Type:            "IssuerService",
//				ServiceEndpoint: serviceUrl,
//			},
//		}
//	}
//	signerPk := getPubKey(pubKeyPath)
//	signerAddr := getAddress(signerPk)
//	signerPrvKey := getPrivateKey(prvKeyPath)
//	singerDid := "did:cnbn:" + signerAddr
//	signature := signDidDocument(didDoc, signerPrvKey)
//	proof := &Proof{
//		Type:               "SM2Signature",
//		Created:            getNowString(),
//		ProofPurpose:       "verificationMethod",
//		VerificationMethod: singerDid + "#keys-1",
//		ProofValue:         signature,
//	}
//
//	proofj, _ := json.Marshal(proof)
//	didDoc.Proof = proofj
//	newDidDocument, _ := json.Marshal(didDoc)
//	return string(newDidDocument)
//}

func getNowString() string {
	currentTime := time.Now().UTC()
	formattedTime := currentTime.Format(time.RFC3339)
	return formattedTime
}

func getPubKeyPem(path string) []byte {
	pem, _ := os.ReadFile(path)
	return pem
}
func getPubKey(pubKeyPath string) crypto.PublicKey {
	pubKey, err := asym.PublicKeyFromPEM(getPubKeyPem(pubKeyPath))
	if err != nil {
		panic(err)
	}
	return pubKey
}
func getPrivateKey(path string) crypto.PrivateKey {
	pem, _ := os.ReadFile(path)
	privKey, err := asym.PrivateKeyFromPEM(pem, nil)
	if err != nil {
		panic(err)
	}
	return privKey
}
func getAddress(pk crypto.PublicKey) string {
	pkBytes, err := evmutils.MarshalPublicKey(pk)
	if err != nil {
		panic(err)
	}
	data := pkBytes[1:]
	bytesAddr := evmutils.Keccak256(data)
	addr := hex.EncodeToString(bytesAddr)[24:]
	return addr
}
func signDidDocument(didDocument *DIDDocument, privKey crypto.PrivateKey) string {
	didDocumentWithoutProof := *didDocument
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

// DIDDocument DID文档
type DIDDocument struct {
	rawData            json.RawMessage
	Context            string   `json:"@context"`
	ID                 string   `json:"id"`
	Controller         []string `json:"controller"`
	Created            string   `json:"created,omitempty"`
	Updated            string   `json:"updated,omitempty"`
	VerificationMethod []struct {
		ID           string `json:"id"`
		PublicKeyPem string `json:"publicKeyPem"`
		Controller   string `json:"controller"`
		Address      string `json:"address"`
	} `json:"verificationMethod"`
	Service []struct {
		ID              string `json:"id"`
		Type            string `json:"type"`
		ServiceEndpoint string `json:"serviceEndpoint"`
	} `json:"service,omitempty"`
	Authentication []string        `json:"authentication"`
	Proof          json.RawMessage `json:"proof,omitempty"`
}

// Proof DID文档或者凭证的证明
type Proof struct {
	Type               string `json:"type"`
	Created            string `json:"created"`
	ProofPurpose       string `json:"proofPurpose"`
	Challenge          string `json:"challenge,omitempty"`
	VerificationMethod string `json:"verificationMethod"`
	ProofValue         string `json:"proofValue,omitempty"`
}

// DocProof DID文档的证明
type DocProof struct {
	Single *Proof
	Array  []*Proof
}

func parseProof(raw json.RawMessage) (DocProof, error) {
	var docProof DocProof
	var single Proof
	var array []*Proof

	if err := json.Unmarshal(raw, &single); err == nil {
		docProof.Single = &single
		return docProof, nil
	}

	if err := json.Unmarshal(raw, &array); err == nil {
		docProof.Array = array
		return docProof, nil
	}

	return docProof, fmt.Errorf("unable to parse proof")
}

// compactJson 压缩json字符串，去掉空格换行等
func compactJson(raw []byte) ([]byte, error) {
	var buf bytes.Buffer
	err := json.Compact(&buf, raw)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// NewDIDDocument 根据DID文档json字符串创建DID文档
func NewDIDDocument(didDocumentJson string) *DIDDocument {
	var didDocument DIDDocument
	err := json.Unmarshal([]byte(didDocumentJson), &didDocument)
	if err != nil {
		return nil
	}
	//fmt.Println("didDocument", didDocument)
	didDocument.rawData = []byte(didDocumentJson)
	return &didDocument
}

// GetProofs 获取DID文档的证明，无论是一个Proof还是多个，都返回数组
func (didDoc *DIDDocument) GetProofs() []*Proof {
	if didDoc.Proof == nil {
		return nil
	}
	docProof, err := parseProof(didDoc.Proof)
	if err != nil {
		return nil
	}
	if docProof.Single != nil {
		return []*Proof{docProof.Single}
	}
	if len(docProof.Array) != 0 {
		return docProof.Array
	}
	return nil
}
