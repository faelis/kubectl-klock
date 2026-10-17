<!--
SPDX-FileCopyrightText: 2026 faelis

SPDX-License-Identifier: CC-BY-4.0
-->

# Spec delta : custom-columns-output

## MODIFIED Requirements

### Requirement: Échappement des séquences terminal

klock MUST neutraliser les séquences d'échappement terminal, les
caractères de contrôle et les caractères invisibles. Cette règle porte sur
les cellules (valeurs comme messages d'erreur), les en-têtes et les
messages d'erreur de l'option `-o`. Les caractères invisibles comprennent
les caractères ignorables par défaut (`Other_Default_Ignorable_Code_Point`).
Ces caractères MUST s'afficher sous une forme visible.

#### Scenario: Annotation avec séquence ANSI

- **WHEN** la colonne est `A:.metadata.annotations.x` et que l'annotation
  `x` vaut l'octet `0x1b` suivi de `[31mrouge`
- **THEN** la cellule affiche `^[[31mrouge`
- **AND** elle ne contient pas l'octet `0x1b`

#### Scenario: Saut de ligne et tabulation

- **WHEN** la colonne est `A:.metadata.annotations.x` et que l'annotation
  `x` vaut `l1`, un saut de ligne, `l2`, une tabulation, `t`
- **THEN** la cellule affiche les six caractères littéraux `l1\nl2\tt`
- **AND** elle ne contient aucun caractère de contrôle

#### Scenario: Autres caractères de contrôle

- **WHEN** l'annotation `x` contient `U+0007`, `U+009B` ou `U+202E`
- **THEN** la cellule affiche respectivement `\a`, `\u009b`, `\u202e`

#### Scenario: Caractères invisibles

- **WHEN** l'annotation `x` contient `U+200B`, `U+2028` ou `U+FEFF`
- **THEN** la cellule affiche respectivement `\u200b`, `\u2028`,
  `\ufeff`

#### Scenario: En-tête avec séquence ANSI

- **WHEN** l'option est `-o custom-columns=` suivie de `X`, de l'octet
  `0x1b`, de `[31m:.metadata.name`
- **THEN** l'en-tête de la colonne est `X^[[31m`
- **AND** il ne contient pas l'octet `0x1b`

#### Scenario: Message d'erreur échappé

- **WHEN** la colonne est `X:.metadata.annotations.x[?(@.a)]` et que
  l'annotation `x` vaut l'octet `0x1b` suivi de `[31m`
- **THEN** la cellule commence par `<error:`
- **AND** elle ne contient pas l'octet `0x1b`

#### Scenario: Caractères ignorables par défaut

- **WHEN** l'annotation `x` contient U+3164, U+115F ou U+034F
- **THEN** la cellule affiche respectivement `\u3164`, `\u115f`,
  `\u034f`

#### Scenario: Spec sans deux-points échappée

- **WHEN** l'option est `-o custom-columns=` suivie de `X` et de l'octet
  `0x1b`
- **THEN** klock s'arrête en erreur
- **AND** le message contient `X^[` et ne contient pas l'octet `0x1b`

#### Scenario: Expression invalide échappée

- **WHEN** l'option est `-o custom-columns=X:.a[` suivie de l'octet
  `0x1b`, de `c]`
- **THEN** klock s'arrête en erreur
- **AND** le message contient `^[c` et ne contient pas l'octet `0x1b`

#### Scenario: Combinaison avec -L échappée

- **WHEN** l'option est `-o custom-columns=A` suivie de l'octet `0x1b` et
  de `:.metadata.name`, avec `-L app`
- **THEN** klock s'arrête en erreur
- **AND** le message ne contient pas l'octet `0x1b`

#### Scenario: Format refusé avec caractère ignorable

- **WHEN** l'option est `-o json` suivie de U+3164
- **THEN** klock s'arrête en erreur
- **AND** le message contient `json\u3164` et ne contient pas U+3164

#### Scenario: En-tête échappé une seule fois dans les erreurs

- **WHEN** l'option est `-o custom-columns=A`, suivie d'un saut de ligne,
  suivie de `:.a[`
- **THEN** klock s'arrête en erreur
- **AND** le message commence par `column "A\n": `, avec un seul
  antislash

## ADDED Requirements

### Requirement: Longueur des cellules bornée

Chaque cellule MUST compter au plus 1024 caractères (points de code) après
échappement et jointure des valeurs multiples. Au-delà, klock MUST
afficher les 1024 premiers suivis de `…`.

#### Scenario: Valeur de 2000 caractères

- **WHEN** la colonne est `A:.metadata.annotations.x` et que l'annotation
  `x` vaut 2000 fois `a`
- **THEN** la cellule `A` affiche 1024 fois `a` suivi de `…`

#### Scenario: Valeurs jointes trop longues

- **WHEN** la colonne est `C:.spec.containers[*].image` et que le pod a
  deux conteneurs dont les images comptent 600 caractères chacune
- **THEN** la cellule `C` compte 1025 caractères et finit par `…`

#### Scenario: Message d'erreur borné

- **WHEN** la colonne est `X:.metadata.annotations.x[?(@.a)]` et que
  l'annotation `x` vaut 5000 fois `a`
- **THEN** la cellule `X` commence par `<error:`, compte 1025 caractères
  et finit par `…`
