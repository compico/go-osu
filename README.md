# osutools


An application for working with OSU.

Features:
- [X] osu database parsing
- [X] Partial parsing of ".osu" bitmap files
- [X] A player for listening to music from osu

## Skills

This package contains beatmap skill analysis algorithms ported from the
[osuSkills](https://github.com/Kert/osuSkills) project.

The implementation was added with the original author's permission:
https://github.com/Kert/osuSkills/issues/5

### About

The purpose of these algorithms is to analyze an osu! beatmap and extract a set
of skill-related metrics, such as:

- Stamina
- Tenacity
- Agility
- Accuracy
- Precision
- Memory
- Reaction
- (and other derived skill values)

Please note that this is a **port** from another programming language. While
care was taken to preserve the original ideas and behavior, some numerical
differences may exist due to implementation details and language-specific
differences.

The primary goal of this package is to preserve the methodology behind the
analysis rather than guarantee bit-for-bit identical results.

### Future changes

The algorithms are expected to evolve over time. As the implementation is
validated and improved, the results may become more accurate or diverge from the
original implementation where appropriate.
