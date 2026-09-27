#!/usr/bin/env python3
"""Unit tests for vlm_probe (V0 of docs/VLM_TEACHER_PLAN.md) - no model, no
network: a fake chat endpoint answers in each coordinate convention."""

import base64
import json
import os
import sys
import tempfile
import unittest

import cv2
import numpy as np

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import vlm_probe as vp  # noqa: E402

CONTACT = (0.40, 0.30, 0.60, 0.70)  # normalised truth the fake model "sees"


def _image_size(payload):
    url = payload["messages"][0]["content"][1]["image_url"]["url"]
    raw = base64.b64decode(url.split(",", 1)[1])
    img = cv2.imdecode(np.frombuffer(raw, np.uint8), cv2.IMREAD_COLOR)
    return img.shape[1], img.shape[0]


def fake_model(mode, fence=True):
    """post_fn answering like a model with the given coordinate mode."""
    def to_model(box, w, h):
        x0, y0, x1, y1 = box
        if mode == "pixel":
            return [round(x0 * w), round(y0 * h), round(x1 * w), round(y1 * h)]
        if mode == "norm1000":
            return [round(v * 1000) for v in box]
        return list(box)

    def post(payload):
        w, h = _image_size(payload)
        prompt = payload["messages"][0]["content"][0]["text"]
        if prompt == vp.CALIB_PROMPT:
            items = [{"label": k, "bbox_2d": to_model(v, w, h)}
                     for k, v in vp.CALIB_BOXES.items()]
        else:
            items = [{"label": "contact", "bbox_2d": to_model(CONTACT, w, h)},
                     {"label": "Thighs", "bbox_2d": to_model((0.0, 0.6, 0.3, 1.0), w, h)}]
        text = json.dumps(items)
        if fence:
            text = "Here you go:\n```json\n" + text + "\n```"
        return {"choices": [{"message": {"content": text}}]}
    return post


class SendSizeTest(unittest.TestCase):
    def test_full_hd_to_896(self):
        self.assertEqual(vp.send_size(1920, 1080), (896, 504))

    def test_multiples_of_28_never_upscaled(self):
        for w, h in ((640, 360), (1280, 720), (720, 1280), (300, 200)):
            sw, sh = vp.send_size(w, h)
            self.assertEqual(sw % 28, 0)
            self.assertEqual(sh % 28, 0)
            self.assertLessEqual(sw, max(w, 28))
            self.assertLessEqual(sh, max(h, 28))
            self.assertLessEqual(max(sw, sh), vp.MAX_SIDE)


class ParseAnswerTest(unittest.TestCase):
    def test_fenced_pixel_answer(self):
        text = '```json\n[{"label": "contact", "bbox_2d": [448, 126, 672, 378]}]\n```'
        status, boxes = vp.parse_answer(text, 1 / 896, 1 / 504)
        self.assertEqual(status, "ok")
        b = boxes[0]
        self.assertEqual(b["label"], "contact")
        self.assertAlmostEqual(b["x0"], 0.5, places=3)
        self.assertAlmostEqual(b["y1"], 0.75, places=3)

    def test_dict_wrapper_swapped_corners_and_clip(self):
        text = 'Sure. {"objects": [{"name": "Brust", "box": [1100, 900, 500, -20]}]}'
        status, boxes = vp.parse_answer(text, 1 / 1000, 1 / 1000)
        self.assertEqual(status, "ok")
        b = boxes[0]
        self.assertEqual(b["label"], "breasts")
        self.assertEqual((b["x0"], b["y0"], b["x1"], b["y1"]), (0.5, 0.0, 1.0, 0.9))

    def test_bad_items_dropped(self):
        text = json.dumps([{"label": "hand", "bbox_2d": [1, 2, 3]},
                           {"label": "hand", "bbox_2d": ["a", 0, 1, 1]},
                           {"label": "hand", "bbox_2d": [0.2, 0.2, 0.2, 0.5]},
                           "junk"])
        self.assertEqual(vp.parse_answer(text, 1, 1), ("empty", []))

    def test_empty_refused_unparsed(self):
        self.assertEqual(vp.parse_answer("[]", 1, 1), ("empty", []))
        self.assertEqual(vp.parse_answer("I'm sorry, I can't help with this image.", 1, 1)[0],
                         "refused")
        self.assertEqual(vp.parse_answer("The image shows two people.", 1, 1)[0], "unparsed")
        self.assertEqual(vp.parse_answer(None, 1, 1)[0], "unparsed")

    def test_labels(self):
        cases = {"contact": "contact", "Penetration": "contact", "left hand": "hand",
                 "hand_2": "hand", "Thighs": "thigh", "leg": "thigh", "eichel": "glans",
                 "kopf": "face", "nipples": "breasts", "sofa": "other", "": "other"}
        for raw, want in cases.items():
            self.assertEqual(vp.normalize_label(raw), want, raw)


class CalibrationTest(unittest.TestCase):
    def test_each_convention_is_recognised(self):
        for mode in vp.NAMED_COORDS:
            res = vp.calibrate("fake", "http://x", 1, post_fn=fake_model(mode), w=896, h=504)
            self.assertTrue(res["ok"], (mode, res))
            self.assertEqual(res["nearest_mode"], mode)
            want = vp.coord_scale(mode, 896, 504)
            self.assertAlmostEqual(res["sx"], want[0], delta=want[0] * 0.02)
            self.assertAlmostEqual(res["sy"], want[1], delta=want[1] * 0.02)

    def test_unusable_answers_fail_closed(self):
        self.assertFalse(vp.fit_calibration("no idea", 896, 504)["ok"])
        one = json.dumps([{"label": "red", "bbox_2d": [70, 60, 340, 262]}])
        self.assertFalse(vp.fit_calibration(one, 896, 504)["ok"])
        # Both boxes present but inconsistent with any single scale.
        bad = json.dumps([{"label": "red", "bbox_2d": [800, 10, 850, 20]},
                          {"label": "blue", "bbox_2d": [10, 400, 60, 480]}])
        self.assertFalse(vp.fit_calibration(bad, 896, 504)["ok"])


class KeyframeTest(unittest.TestCase):
    def test_interval_plus_scene_midpoints_deduplicated(self):
        t = vp.keyframe_times_ms(20000, 5.0, [(0, 8000), (8000, 20000)])
        self.assertEqual(t, [2500, 4000, 7500, 12500, 14000, 17500])

    def test_no_interval(self):
        self.assertEqual(vp.keyframe_times_ms(10000, 0, [(0, 10000)]), [5000])


class ProbeVideoTest(unittest.TestCase):
    def _video(self, d):
        path = os.path.join(d, "clip.avi")
        vw = cv2.VideoWriter(path, cv2.VideoWriter_fourcc(*"MJPG"), 10, (320, 180))
        for i in range(60):
            f = np.full((180, 320, 3), 40 + i, np.uint8)
            vw.write(f)
        vw.release()
        return path

    def test_end_to_end_with_auto_calibration(self):
        for mode in vp.NAMED_COORDS:
            with tempfile.TemporaryDirectory() as d:
                video = self._video(d)
                ov = os.path.join(d, "ov")
                res = vp.probe_video(video, "fake", every_s=2.0, scenes=False,
                                     post_fn=fake_model(mode, fence=(mode != "pixel")),
                                     overlay_dir=ov, log=lambda *_: None)
                self.assertEqual(res["calibration"]["nearest_mode"], mode)
                self.assertEqual((res["sent_w"], res["sent_h"]), (308, 168))
                self.assertEqual([f["t_ms"] for f in res["frames"]], [1000, 3000, 5000])
                for f in res["frames"]:
                    self.assertEqual(f["status"], "ok")
                    c = [b for b in f["boxes"] if b["label"] == "contact"][0]
                    for got, want in zip((c["x0"], c["y0"], c["x1"], c["y1"]), CONTACT):
                        self.assertAlmostEqual(got, want, delta=0.01, msg=mode)
                    self.assertIn("thigh", [b["label"] for b in f["boxes"]])
                s = res["summary"]
                self.assertEqual(s["status"], {"ok": 3})
                self.assertEqual(s["contact_rate"], 1.0)
                self.assertEqual(s["refusal_rate"], 0.0)
                self.assertEqual(len(os.listdir(ov)), 3)
                json.dumps(res)  # the output must be serialisable

    def test_failed_calibration_stops_before_frames(self):
        calls = []

        def post(payload):
            calls.append(payload)
            return {"choices": [{"message": {"content": "sorry, I can't"}}]}
        with tempfile.TemporaryDirectory() as d:
            with self.assertRaises(RuntimeError):
                vp.probe_video(self._video(d), "fake", scenes=False, post_fn=post,
                               log=lambda *_: None)
        self.assertEqual(len(calls), 1)

    def test_errors_and_refusals_are_recorded_not_fatal(self):
        answers = iter(["sorry, this is explicit content", RuntimeError("timeout"), "[]"])

        def post(payload):
            a = next(answers)
            if isinstance(a, Exception):
                raise a
            return {"choices": [{"message": {"content": a}}]}
        with tempfile.TemporaryDirectory() as d:
            res = vp.probe_video(self._video(d), "fake", every_s=2.0, scenes=False,
                                 coord="norm1", post_fn=post, log=lambda *_: None)
        self.assertEqual([f["status"] for f in res["frames"]], ["refused", "error", "empty"])
        self.assertAlmostEqual(res["summary"]["refusal_rate"], 0.333, places=3)


if __name__ == "__main__":
    unittest.main()
