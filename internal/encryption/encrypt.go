package encryption

import (
    "bytes"
    "fmt"

    "golang.org/x/crypto/openpgp"
)

// Encrypt chiffre le message avec la clé publique PGP (format ASCII-armored ou binaire).
// v1: implémentation simple, sans gestion avancée des préférences.
func Encrypt(pubKey string, raw []byte) ([]byte, error) {
    entityList, err := readPubKey(pubKey)
    if err != nil {
        return nil, fmt.Errorf("pgp: invalid public key: %w", err)
    }

    var buf bytes.Buffer
    w, err := openpgp.Encrypt(&buf, entityList, nil, nil, nil)
    if err != nil {
        return nil, fmt.Errorf("pgp: encrypt: %w", err)
    }

    if _, err := w.Write(raw); err != nil {
        return nil, fmt.Errorf("pgp: write: %w", err)
    }
    if err := w.Close(); err != nil {
        return nil, fmt.Errorf("pgp: close: %w", err)
    }

    return buf.Bytes(), nil
}

func readPubKey(armored string) (openpgp.EntityList, error) {
    r := bytes.NewBufferString(armored)

    // On tente d'abord ASCII-armored, puis binaire.
    el, err := openpgp.ReadArmoredKeyRing(r)
    if err == nil {
        return el, nil
    }

    r2 := bytes.NewBufferString(armored)
    el, err = openpgp.ReadKeyRing(r2)
    if err != nil {
        return nil, err
    }
    return el, nil
}
