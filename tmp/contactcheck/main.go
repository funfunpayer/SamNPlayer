package main
import (
  "encoding/json"
  "fmt"
  "os"
  "github.com/funfunpayer/SamNPlayer/funscript"
)
func main() {
  data, err := os.ReadFile(os.Args[1])
  if err != nil { panic(err) }
  var s funscript.Script
  if err := json.Unmarshal(data, &s); err != nil { panic(err) }
  opts := funscript.MapOptionsFromScript(&s)
  frames := s.ToIntensityCurve(opts)
  var maxV, maxS float64
  var nVib int
  for _, f := range frames {
    if f.Vibration > maxV { maxV = f.Vibration }
    if f.Suction > maxS { maxS = f.Suction }
    if f.Vibration > 0.05 { nVib++ }
  }
  fmt.Printf("sync=%v contact=%v frames=%d maxVib=%.3f maxSuc=%.3f vibActiveFrames=%d (%.1f%%)\n",
    opts.Sync, opts.ContactVibration, len(frames), maxV, maxS, nVib, 100*float64(nVib)/float64(len(frames)+1))
}
