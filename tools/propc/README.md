# propc — static models as code

Buildings, base walls and gates, outposts and siege machines described in Python from plain shapes
(`art/props/*.py`), written as the game's static models (`frontend/public/models/<id>/lod0..2.glb`
+ `model.json`, like `tools/meshbake`). Needs numpy and pillow (and `tools/modelc` for `render`).

```sh
python3 tools/propc/propc list
python3 tools/propc/propc build            # everything (or: build keep tunnel)
python3 tools/propc/propc render keep      # art/build/keep/render.png
```

Design and conventions: [`docs/MODEL_PIPELINE.md`](../../docs/MODEL_PIPELINE.md), section 9.
