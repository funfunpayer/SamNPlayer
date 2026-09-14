#!/usr/bin/env python3
"""generate_funscript.py - einfacher, klassischer CV-basierter funscript-Generator.

Kein Deep Learning, keine trainierten Modelle - der Nutzer markiert per Hand
eine Bildregion (ROI) im ersten Frame, ein OpenCV-Tracker (CSRT) verfolgt sie
durchs Video, die vertikale Bewegung wird zu einer funscript-Positionskurve.

Das ist bewusst der "einfache" Ansatz (vergleichbar mit dem, was in dieser
Nische vor Deep-Learning-Ansätzen üblich war: manuelle ROI + klassischer
Tracker) - eigenständig geschrieben, keine Codeübernahme von irgendwo.
Für komplexe/verdeckte Szenen ist ein trainiertes Objekterkennungsmodell
(YOLO o.ä.) deutlich robuster, das ist hier bewusst nicht das Ziel.

Pipeline:
  1. ROI im ersten Frame per CSRT-Tracker durchs Video verfolgen -> rohe
     y-Positionskurve.
  1b. Camera Motion Compensation: globale Kamerabewegung wird aus
      Hintergrund-Features (außerhalb der ROI) per Sparse Optical Flow +
      robuster affiner Schätzung (RANSAC) gemessen und von der Rohkurve
      abgezogen, damit ein Kameraschwenk nicht als Objektbewegung
      fehlinterpretiert wird (kann per --no-camera-compensation abgeschaltet
      werden).
  2. Savitzky-Golay-Filter zur Rauschunterdrückung.
  3. Min/Max-Normalisierung auf 0-100 (funscript-Konvention: 100 = "oben").
  4. Peak/Valley-Erkennung zur Reduktion auf sinnvolle Keyframes (statt jeden
     Frame als Punkt zu speichern - das wäre unnötig groß und ruckelig).
  4b. Optionale RDP-Simplifizierung (--rdp-tolerance) als zusätzliche
      Redundanzreduktion auf den bereits gefundenen Keyframes.
  5. Schreiben als .funscript (JSON, {"actions": [{"at": ms, "pos": 0-100}]}).

Nutzung:
  python3 generate_funscript.py --video input.mp4 --roi "120,80,60,60" \\
      --output out.funscript

Ohne --roi öffnet sich ein interaktives Auswahlfenster (cv2.selectROI) auf
dem ersten Frame - erfordert eine lokale Anzeige (kein SSH ohne X-Forwarding).
"""
