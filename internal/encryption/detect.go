package encryption

import "strings"

// IsPGPEncrypted détecte grossièrement si le message est déjà chiffré PGP.
func IsPGPEncrypted(raw []byte) bool {
    s := string(raw)

    if strings.Contains(s, "-----BEGIN PGP MESSAGE-----") {
        return true
    }
    if strings.Contains(s, "Content-Type: multipart/encrypted") {
        return true
    }
    if strings.Contains(s, "Content-Type: application/pgp-encrypted") {
        return true
    }

    return false
}
