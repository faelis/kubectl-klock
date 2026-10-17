<!--
SPDX-FileCopyrightText: 2026 faelis

SPDX-License-Identifier: CC-BY-4.0
-->

# Spec delta : table-output

## Purpose

Afficher en direct la représentation `Table` que renvoie le serveur
(colonnes par défaut, `-o wide`, colonnes `-L`) sans laisser passer de
séquence terminal ni de caractère invisible.

## ADDED Requirements

### Requirement: Échappement des cellules en mode table

klock MUST neutraliser les séquences d'échappement terminal, les
caractères de contrôle et les caractères invisibles. Cette règle porte sur
chaque cellule texte, liste ou objet de la `Table` du serveur, avec les
règles du mode custom-columns. Les cellules numériques, booléennes ou
nulles MUST s'afficher comme avant. La coloration des statuts et des
fractions MUST porter sur la valeur échappée.

#### Scenario: Message d'Event avec OSC 52

- **WHEN** klock surveille `events` et que la cellule `MESSAGE` vaut
  l'octet `0x1b`, puis `]52;c;aGVsbG8=`, puis l'octet `0x07`
- **THEN** la cellule affiche `^[]52;c;aGVsbG8=\a`
- **AND** elle ne contient ni l'octet `0x1b` ni l'octet `0x07`

#### Scenario: Saut de ligne dans une colonne de CR

- **WHEN** une colonne de la `Table` vaut `l1`, un saut de ligne, `l2`
- **THEN** la cellule affiche les six caractères littéraux `l1\nl2`

#### Scenario: Caractère invisible en mode table

- **WHEN** une cellule vaut `a`, U+200B, `b`
- **THEN** la cellule affiche `a\u200bb`

#### Scenario: Cellule liste

- **WHEN** une cellule est une liste qui contient la chaîne formée de
  l'octet `0x1b` et de `x`
- **THEN** la cellule contient `^[x`
- **AND** elle ne contient pas l'octet `0x1b`

#### Scenario: Valeur numérique inchangée

- **WHEN** une colonne de type `integer` vaut `3`
- **THEN** la cellule affiche `3`

#### Scenario: Statut toujours coloré

- **WHEN** la cellule `STATUS` d'un pod vaut `Running`
- **THEN** la cellule affiche `Running` avec le style du statut `Running`

### Requirement: Échappement des en-têtes en mode table

klock MUST appliquer le même échappement aux en-têtes tirés des
définitions de colonnes du serveur et des clés `-L`, après leur passage en
majuscules.

#### Scenario: Nom de colonne avec séquence ANSI

- **WHEN** une définition de colonne s'appelle `x`, suivi de l'octet
  `0x1b`, suivi de `[31m`
- **THEN** l'en-tête de la colonne est `X^[[31M`
- **AND** il ne contient pas l'octet `0x1b`

#### Scenario: Échappement après les majuscules

- **WHEN** une définition de colonne s'appelle `x` suivi de U+202E
- **THEN** l'en-tête de la colonne est `X\u202e`

#### Scenario: Clé de label avec séquence ANSI

- **WHEN** l'option est `-L` suivie de `app` et de l'octet `0x1b`
- **THEN** l'en-tête de la colonne est `APP^[`

### Requirement: Échappement des colonnes de labels

klock MUST échapper la valeur affichée dans chaque colonne `-L`.

#### Scenario: Valeur de label avec séquence ANSI

- **WHEN** l'option est `-L app` et que l'objet reçu porte le label `app`
  égal à l'octet `0x1b` suivi de `[31m`
- **THEN** la cellule `APP` affiche `^[[31m`
- **AND** elle ne contient pas l'octet `0x1b`

### Requirement: Longueur des cellules bornée en mode table

Chaque cellule texte MUST compter au plus 1024 caractères (points de code)
après échappement. Au-delà, klock MUST afficher les 1024 premiers suivis
de `…`.

#### Scenario: Cellule de 2000 caractères

- **WHEN** une cellule vaut 2000 fois `a`
- **THEN** la cellule affiche 1024 fois `a` suivi de `…`

#### Scenario: Cellule de 1024 caractères

- **WHEN** une cellule vaut 1024 fois `a`
- **THEN** la cellule affiche 1024 fois `a`, sans `…`

#### Scenario: Borne après échappement

- **WHEN** une cellule vaut 1000 fois `a` suivi de 100 sauts de ligne
- **THEN** la cellule affiche 1000 fois `a`, puis 12 fois `\n`, puis
  `…`
