# Acknowledgements

## Clef decision models

Dessau's tests hold its decision-model support to the behaviour of the
reference implementation, through golden fixtures recorded from it. The
fixtures in `internal/runtime/testdata/clef/` are the reference's answers to a
fixed set of generic requests, recorded on an Apple Silicon Mac by running the
reference server outside Dessau.

- **The reference implementation**: `clef_mlx.py`, the loader and server
  shipped in the
  [mlx-community/clef-flash-4bit](https://huggingface.co/mlx-community/clef-flash-4bit)
  and [mlx-community/clef-4bit](https://huggingface.co/mlx-community/clef-4bit)
  repositories, licensed under the
  [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0). Dessau
  does not ship or import it; Dessau's own decision server is written against
  its behaviour.
- **The models**: [Cloudflare/clef-flash](https://huggingface.co/Cloudflare/clef-flash)
  (built on Qwen/Qwen3.5-9B) and [Cloudflare/clef](https://huggingface.co/Cloudflare/clef)
  (fine-tuned from Qwen/Qwen3.8-27B), both Apache-2.0, in their MLX
  conversions by [mlx-community](https://huggingface.co/mlx-community).

Each fixture set's `MANIFEST.json` names the model repository and commit it was
recorded against, the sha256 of the model files, and the versions of mlx,
mlx-lm, mlx-vlm and Python used.
