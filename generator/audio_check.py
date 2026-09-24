"""audio_check.py - Plausibilitätsprüfung des Skript-Tempos gegen die
Audiospur des Videos.

Kein KI-Baustein (gehört deshalb nicht zu docs/AI_ADAPTER.md), sondern eine
klassische, regelbasierte Prüfung im Stil von quality_doctor.py: beide
messen eine Zahl aus dem Signal und vergleichen sie gegen eine Erwartung,
ohne trainiertes Modell.

Idee: die dominante Frequenz der Audio-Energiehüllkurve (wie laut/aktiv ist
die Tonspur gerade) korreliert oft mit dem tatsächlichen Bewegungstempo -
nicht 1:1 phasengleich wie eine Positionskurve, aber im TEMPO vergleichbar.
Weicht das aus dem Video getrackte Skript-Tempo stark vom Audio-Tempo ab
(auch nicht bei einem plausiblen Vielfachen wie x2/x0.5), ist das ein
Hinweis - kein Beweis -, dass das Tracking danebenliegt oder eine
untypische Szene vorliegt. Genau wie ai_quality.py's Zweitmeinung: die
Prüfung liefert eine WARNUNG, sie überschreibt nie quality["passed"]/
quality["score"] und korrigiert nichts automatisch.

Ausführen der Tests: python3 generator/audio_check_test.py
"""

import subprocess
import sys

import numpy as np

# Plausibler Bereich für ein Bewegungstempo - siehe device_profile.py
# (SLOWEST_FULL_STROKE_MS entspricht ~1.1Hz als langsamste sinnvolle Grenze;
# 5Hz ist bereits deutlich schneller als jede reale Vollhub-Bewegung).
MIN_TEMPO_HZ = 0.2
MAX_TEMPO_HZ = 5.0

# Erlaubte Verhältnisse zwischen Skript- und Audio-Tempo. Ein Ton pro Hub UND
# Rückzug (statt nur pro Hub) macht das Audio-Tempo doppelt so hoch wie das
# Skript-Tempo - das ist kein Fehler, sondern eine andere, ebenso plausible
# Zählweise.
DEFAULT_HARMONICS = (0.5, 1.0, 2.0)


class AudioUnavailable(Exception):
    """ffmpeg fehlt oder das Video hat keine (lesbare) Audiospur."""


def _dominant_frequency_hz(values, sample_rate_hz, min_hz=MIN_TEMPO_HZ, max_hz=MAX_TEMPO_HZ):
    """FFT-Spitzenfrequenz innerhalb eines plausiblen Tempobereichs. Reine
    Funktion, unabhängig davon ob values aus Video oder Audio stammt."""
    values = np.asarray(values, dtype=float)
    n = len(values)
    if n < 8:
        return None
    signal = values - np.mean(values)
    spectrum = np.abs(np.fft.rfft(signal))
    freqs = np.fft.rfftfreq(n, d=1.0 / sample_rate_hz)
    mask = (freqs >= min_hz) & (freqs <= max_hz)
    if not np.any(mask) or not np.any(spectrum[mask] > 0):
        return None
    band_spectrum = spectrum[mask]
    band_freqs = freqs[mask]
    return float(band_freqs[np.argmax(band_spectrum)])


def audio_envelope(samples, sample_rate, window_ms=50.0):
    """Kurzzeit-Energiehüllkurve (RMS je Fenster). Macht aus dem
    hochfrequenten Rohsignal (typisch mehrere kHz) eine Kurve im Bereich der
    Bewegungsrate (<5Hz), auf der sich überhaupt erst eine Tempo-FFT lohnt.
    Gibt (envelope, envelope_rate_hz) zurück; envelope ist leer, wenn
    samples zu kurz für mindestens zwei Fenster sind."""
    samples = np.asarray(samples, dtype=float)
    window = max(1, int(round(sample_rate * window_ms / 1000.0)))
    n_windows = len(samples) // window
    if n_windows < 2:
        return np.zeros(0), 0.0
    trimmed = samples[:n_windows * window].reshape(n_windows, window)
    rms = np.sqrt(np.mean(trimmed ** 2, axis=1))
    return rms, 1000.0 / window_ms


def estimate_audio_tempo_hz(samples, sample_rate, window_ms=50.0):
    """Dominantes Tempo der Audio-Energiehüllkurve, in Hz. None, wenn das
    Signal zu kurz oder zu leise/gleichmäßig für eine belastbare Aussage
    ist - dann bleibt die Prüfung aus, statt eine Zufallszahl zu vergleichen."""
    envelope, envelope_rate = audio_envelope(samples, sample_rate, window_ms=window_ms)
    if len(envelope) < 8:
        return None
    return _dominant_frequency_hz(envelope, envelope_rate)


def estimate_script_tempo_hz(actions):
    """Dominantes Tempo der Skript-eigenen Positionskurve. Wie
    quality_doctor.py's Rhythmus-Messung auf ein gleichmäßiges Zeitraster
    resampled (FFT braucht das), da Skript-Actions nicht äquidistant sind."""
    if len(actions) < 8:
        return None
    at = np.array([a["at"] for a in actions], dtype=float)
    pos = np.array([a["pos"] for a in actions], dtype=float)
    order = np.argsort(at)
    at, pos = at[order], pos[order]
    if at[-1] - at[0] <= 0:
        return None
    step_ms = max(float(np.median(np.diff(at))), 10.0)
    grid = np.arange(at[0], at[-1], step_ms)
    if len(grid) < 8:
        return None
    pos_uniform = np.interp(grid, at, pos)
    return _dominant_frequency_hz(pos_uniform, 1000.0 / step_ms)


def compare_tempo(script_hz, audio_hz, tolerance=0.25, harmonics=DEFAULT_HARMONICS):
    """Prüft, ob script_hz zu EINEM der erlaubten Vielfachen von audio_hz
    passt (siehe DEFAULT_HARMONICS). Gibt None zurück, wenn eine der beiden
    Zahlen fehlt (keine Aussage statt eines falschen Vergleichs), sonst ein
    dict mit dem besten Vielfachen und ob es innerhalb der Toleranz liegt."""
    if script_hz is None or audio_hz is None or script_hz <= 0 or audio_hz <= 0:
        return None
    best_harmonic = min(harmonics, key=lambda h: abs(script_hz - audio_hz * h))
    predicted_hz = audio_hz * best_harmonic
    relative_error = abs(script_hz - predicted_hz) / script_hz
    return {
        "matches": relative_error <= tolerance,
        "harmonic": best_harmonic,
        "predicted_hz": predicted_hz,
        "relative_error": relative_error,
    }


def extract_audio_samples(video_path, sample_rate=8000):
    """Dekodiert die Audiospur über ffmpeg zu Mono-Float32-Rohsamples (kein
    Zwischenformat wie WAV nötig - ffmpeg schreibt das Rohsignal direkt auf
    stdout). Braucht ein installiertes ffmpeg auf dem PATH; wirft
    AudioUnavailable statt abzustürzen, wenn ffmpeg fehlt oder das Video
    keine (lesbare) Audiospur hat."""
    cmd = ["ffmpeg", "-i", str(video_path), "-vn", "-ac", "1", "-ar", str(sample_rate),
           "-f", "f32le", "-loglevel", "error", "-"]
    try:
        proc = subprocess.run(cmd, capture_output=True)
    except FileNotFoundError:
        raise AudioUnavailable("ffmpeg ist nicht installiert")
    if proc.returncode != 0 or len(proc.stdout) == 0:
        detail = proc.stderr.decode("utf-8", "replace").strip()[:200]
        raise AudioUnavailable(f"Keine Audiospur lesbar: {detail or 'unbekannter ffmpeg-Fehler'}")
    return np.frombuffer(proc.stdout, dtype=np.float32), sample_rate


def check(video_path, actions, sample_rate=8000, window_ms=50.0, tolerance=0.25):
    """Vergleicht Skript- und Audio-Tempo und liefert eine Warnliste im
    Stil von quality_doctor.evaluate(). PLAUSIBILITÄTSPRÜFUNG, keine
    Korrektur - eine Abweichung kann echte, aber untypische Bewegung sein
    (ruhige Szene, Stille) oder ein Tracking-Fehler; beides braucht einen
    Menschen zur Einordnung, keine automatische Reaktion."""
    script_hz = estimate_script_tempo_hz(actions)
    try:
        samples, sr = extract_audio_samples(video_path, sample_rate=sample_rate)
    except AudioUnavailable as exc:
        return {"available": False, "reason": str(exc), "warnings": []}

    audio_hz = estimate_audio_tempo_hz(samples, sr, window_ms=window_ms)
    comparison = compare_tempo(script_hz, audio_hz, tolerance=tolerance)

    warnings = []
    if comparison is not None and not comparison["matches"]:
        warnings.append(
            f"Script tempo ({script_hz:.2f} Hz) does not match an expected multiple "
            f"of audio tempo ({audio_hz:.2f} Hz; nearest {comparison['harmonic']}x "
            f"→ {comparison['predicted_hz']:.2f} Hz) — may be real atypical motion "
            "or a tracking error; no automatic correction")

    return {
        "available": True,
        "script_hz": script_hz,
        "audio_hz": audio_hz,
        "comparison": comparison,
        "warnings": warnings,
    }


def append_quality_audio_hint(quality_passed: bool, result: dict) -> dict:
    """G1.2: when Signal Quality failed and audio Hz is clear, add English hint.
    Never changes actions — warn only. Mutates and returns result."""
    if not isinstance(result, dict):
        return result
    if quality_passed or not result.get("available"):
        return result
    audio_hz = result.get("audio_hz")
    if audio_hz is None:
        return result
    msg = (
        f"Audio suggests ~{audio_hz:.2f} Hz — check ROI / axis "
        "(Signal Quality failed; audio did not rewrite the curve)"
    )
    warnings = list(result.get("warnings") or [])
    if msg not in warnings:
        warnings.append(msg)
    result["warnings"] = warnings
    return result


def main():
    import argparse
    import json

    ap = argparse.ArgumentParser(description=__doc__,
                                  formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--video", required=True)
    ap.add_argument("--funscript", required=True, help="Zu prüfende .funscript-Datei")
    ap.add_argument("--tolerance", type=float, default=0.25)
    args = ap.parse_args()

    with open(args.funscript) as f:
        actions = json.load(f)["actions"]

    result = check(args.video, actions, tolerance=args.tolerance)
    if not result["available"]:
        print(f"Audio-Prüfung nicht möglich: {result['reason']}", file=sys.stderr)
        sys.exit(0)
    print(f"Skript-Tempo: {result['script_hz']}, Audio-Tempo: {result['audio_hz']}",
          file=sys.stderr)
    for w in result["warnings"]:
        print(f"WARNUNG: {w}", file=sys.stderr)


if __name__ == "__main__":
    main()
