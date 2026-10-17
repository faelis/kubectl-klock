<!--
SPDX-FileCopyrightText: 2026 faelis

SPDX-License-Identifier: CC-BY-4.0
-->

# Proposal : sanitize-table-output

## Why

En mode table, klock affiche les cellules de la `Table` du serveur sans les
échapper. Un tenant peut écrire une séquence OSC 52 dans le message d'un
Event : quand l'admin lance `klock events`, son presse-papiers est
remplacé (constat C1 de `cso-1.md`, change `add-custom-columns`). Le mode
custom-columns, lui, échappe ses cellules, ce qui rend l'écart visible.
L'audit relève aussi trois points Low (C2 à C4) et une note (C5), traités
ici.

## What Changes

- Mode table : les cellules texte, les en-têtes et les colonnes `-L`
  passent par l'échappement du mode custom-columns (C1).
- L'échappement couvre aussi les caractères
  `Other_Default_Ignorable_Code_Point` (U+3164, U+115F, U+034F…), dans
  les deux modes (C3).
- Les messages d'erreur de l'option `-o` sont échappés : spec ou fichier
  invalide, combinaison avec `-L` (C2).
- Chaque cellule est bornée à 1024 caractères suivis de `…`, dans les
  deux modes (C4).
- Suggestions du filtre et cellule `NAMESPACE` échappées : les noms de
  Role et de ClusterRole acceptent les caractères de contrôle (constat N1
  de `reviews/cso-1.md`).
- README : la phrase sur l'échappement précise que les sélecteurs de
  variante restent intacts, pour les emoji (C5).

## Capabilities

### New Capabilities

- `table-output` : affichage en direct de la `Table` du serveur (colonnes
  par défaut, `-o wide`, `-L`) sans séquence terminal brute.

### Modified Capabilities

- `custom-columns-output` : échappement étendu aux caractères ignorables
  et aux messages d'erreur de `-o` ; longueur des cellules bornée.

## Impact

- `pkg/klock/klock.go` : `addObjectToTable`, `updateColDefHeaders`,
  `labelColumnHeader`, `sanitizeCell`, `evalCell`, `Options.Validate`.
- `pkg/klock/customcolumns.go` : messages d'erreur de
  `ParseCustomColumns`.
- `README.md` : section « Custom columns ».
- Aucune dépendance ajoutée, aucune option ajoutée.

## Taille

S : un seul paquet, une fonction d'échappement déjà en place réutilisée,
aucune brique ni décision d'architecture nouvelle.

## Hors périmètre

- Sélecteurs de variante (U+FE0E, U+FE0F, U+E0100…) : les échapper
  casserait l'affichage des emoji.
- Erreurs du watch ou du serveur affichées par `Table.SetError`.
- Pagination de la liste initiale.
- Longueur maximale configurable.
