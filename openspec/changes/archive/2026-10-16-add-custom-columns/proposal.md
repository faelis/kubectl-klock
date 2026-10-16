<!--
SPDX-FileCopyrightText: 2026 faelis

SPDX-License-Identifier: CC-BY-4.0
-->

# Proposal

## Why

`kubectl get -o custom-columns=…` laisse l'utilisateur choisir ses colonnes
par JSONPath, mais klock refuse ce format. Pour surveiller un champ absent de
la table du serveur (image d'un conteneur, nœud, annotation), il faut donc
revenir à `kubectl get --watch` et perdre l'interface de klock.

## What Changes

- `-o custom-columns=SPEC` est accepté. La syntaxe est celle de kubectl :
  `EN-TÊTE:expression-jsonpath`, séparés par des virgules.
- `-o custom-columns-file=CHEMIN` est accepté. Le fichier suit le format de
  kubectl : en-têtes sur la première ligne, expressions sur la seconde.
- En mode custom-columns, klock demande les objets complets au serveur
  plutôt que sa table, et calcule chaque cellule à partir de l'objet.
- Le tri, le filtre, `--hide-deleted` et la coloration selon la valeur
  marchent dans ce mode.
- Combiner custom-columns et `-L` produit une erreur, comme dans kubectl.
- La complétion de `-o` propose `custom-columns=` et `custom-columns-file=`.

## Capabilities

### New Capabilities

- `custom-columns-output` : affichage en direct des ressources selon des
  colonnes définies par l'utilisateur en JSONPath (spec inline ou fichier).

### Modified Capabilities

Aucune : le dépôt n'a pas encore de spec.

## Impact

- Code : `pkg/klock/klock.go` (options, requête, `Printer`),
  `pkg/klock/customcolumns.go` (nouveau : lecture de la spec), `cmd/root.go`
  (aide, exemples, complétion).
- Dépendances : aucune nouvelle. klock importe en plus
  `k8s.io/kubectl/pkg/cmd/get` et `k8s.io/client-go/util/jsonpath`, deux
  modules déjà requis par `go.mod`.
- Documentation : README, section sur les formats de sortie.
- Taille : S. Le changement reste dans un paquet et touche trois fichiers de
  code. Aucun composant ni interface externe nouveau.

## Hors périmètre

- Les autres formats de kubectl : `json`, `yaml`, `jsonpath`,
  `go-template`, `name`.
- `--no-headers` : klock affiche toujours les en-têtes.
- `--sort-by` sur une colonne personnalisée.
- La mise en forme spéciale des horodatages (âge relatif) dans les colonnes
  personnalisées.
- L'ajout automatique d'une colonne `NAMESPACE` avec `-A`.
