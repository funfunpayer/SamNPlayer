package featurepreview

import (
 "errors"
 "testing"
 "time"
)

var epoch = time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)

func TestSevenDayBoundary(t *testing.T) {
 s, err := Activate("ai-repair-preview", epoch)
 if err != nil { t.Fatal(err) }
 cases := []struct { at time.Time; remaining time.Duration; want error }{
  {epoch, Duration, nil},
  {epoch.Add(6*24*time.Hour), 24*time.Hour, nil},
  {epoch.Add(Duration-time.Nanosecond), time.Nanosecond, nil},
  {epoch.Add(Duration), 0, ErrExpired},
  {epoch.Add(Duration+time.Second), 0, ErrExpired},
 }
 for _, tc := range cases {
  _, remaining, err := Check(s, tc.at)
  if !errors.Is(err, tc.want) || remaining != tc.remaining {
   t.Errorf("at=%v remaining=%v error=%v; want %v %v",tc.at,remaining,err,tc.remaining,tc.want)
  }
 }
}
func TestStartOnlyOnExplicitActivation(t *testing.T) {
 if _,_,err:=Gate("ai-repair-preview",nil,epoch); !errors.Is(err,ErrNotStarted){t.Fatal(err)}
 if _,err:=Activate("",epoch);!errors.Is(err,ErrInvalidState){t.Fatal(err)}
}
func TestNoRenewalOnRepeatedGate(t *testing.T) {
 s,_:=Activate("feature-a",epoch)
 s,_,_ = Gate("feature-a",&s,epoch.Add(4*24*time.Hour))
 _,_,err:=Gate("feature-a",&s,epoch.Add(Duration+time.Second))
 if !errors.Is(err,ErrExpired){t.Fatal(err)}
 if !s.StartedAt.Equal(epoch){t.Fatal("gate changed activation time")}
}
func TestClockRollback(t *testing.T) {
 s,_:=Activate("feature-a",epoch)
 s,_,_ = Check(s,epoch.Add(48*time.Hour))
 _,_,err:=Check(s,epoch.Add(24*time.Hour))
 if !errors.Is(err,ErrClockRollback){t.Fatal(err)}
}
func TestFeatureIsolation(t *testing.T) {
 s,_:=Activate("one",epoch)
 if _,_,err:=Gate("two",&s,epoch); !errors.Is(err,ErrInvalidState){t.Fatal(err)}
}
func TestCorruptedState(t *testing.T) {
 for _,s :=range []State{{},{FeatureID:"one",StartedAt:epoch},{FeatureID:"one",StartedAt:epoch,LastSeenAt:epoch.Add(-time.Second)}} {
  if _,_,err:=Check(s,epoch);!errors.Is(err,ErrInvalidState){t.Fatalf("%+v: %v",s,err)}
 }
}
func TestTimezoneNormalization(t *testing.T) {
 zone:=time.FixedZone("local",2*3600)
 start:=epoch.In(zone)
 s,_:=Activate("test",start)
 if !s.StartedAt.Equal(epoch)||s.StartedAt.Location()!=time.UTC{t.Fatal(s)}
}
