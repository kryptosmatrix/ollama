package instructionsgolden_test
import (
 "bytes"
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "os"
 "path/filepath"
 "testing"
)
type canonicalProfile struct {
 SchemaVersion int `json:"schema_version"`
 Revision string `json:"revision"`
 Enabled bool `json:"enabled"`
 Text string `json:"text"`
 Binding json.RawMessage `json:"binding"`
}
func fixture(t *testing.T, name string) []byte {
 t.Helper()
 data, err := os.ReadFile(filepath.Join(os.Getenv("G1_GOLDEN_ROOT"),name+".json"))
 if err != nil {t.Fatal(err)}
 return data
}
func TestG1CanonicalProfileGoldenBytes(t *testing.T) {
 for _, c := range []struct{name,hash string}{
{"zero","157826516a6fd8d118dc84113f5d5635604ba0fd871d98412816611627e43b36"},
{"unicode","bf623f85cbc81809c618dbe0fa0767901ea22449804e2dc392079d22d27ac984"},
{"escapes","d11cedadb67f8a65b05ee10f15fe6d7342218530dd2b93e13ff7f4accfa4d104"},
 } {
  t.Run(c.name,func(t *testing.T){
   expected := fixture(t,c.name)
   var value canonicalProfile
   if err:=json.Unmarshal(expected,&value);err!=nil {t.Fatal(err)}
   actual,err:=json.Marshal(value);if err!=nil {t.Fatal(err)}
   if !bytes.Equal(actual,expected){t.Fatalf("canonical byte mismatch: got %q want %q",actual,expected)}
   digest:=sha256.Sum256(actual)
   if hex.EncodeToString(digest[:])!=c.hash {t.Fatal("independent Python digest differs from Go digest")}
   value.Text+=" altered"
   changed,err:=json.Marshal(value);if err!=nil {t.Fatal(err)}
   if sha256.Sum256(changed)==digest {t.Fatal("changed text failed to change the canonical digest")}
   if bytes.HasSuffix(actual,[]byte("\n")){t.Fatal("unexpected trailing newline")}
  })
 }
}
func TestG1CanonicalProfileDistinguishesLineEndings(t *testing.T){
 a:=canonicalProfile{SchemaVersion:1,Revision:"2",Enabled:true,Text:"a\nb"}
 b:=a;b.Text="a\r\nb"
 x,err:=json.Marshal(a);if err!=nil{t.Fatal(err)}
 y,err:=json.Marshal(b);if err!=nil{t.Fatal(err)}
 if bytes.Equal(x,y)||sha256.Sum256(x)==sha256.Sum256(y){t.Fatal("distinct line endings normalised")}
}
