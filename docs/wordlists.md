# Wordlists

`controlplane/login_key_words.txt` is the English BIP-39 wordlist from the
Bitcoin Improvement Proposals repository:

- source: `https://github.com/bitcoin/bips/blob/master/bip-0039/english.txt`
- license: MIT, as stated by BIP-0039

Shellin uses this list only to generate short, human-transcribed one-time login
keys for device login. These keys are not cryptocurrency mnemonic phrases and
are not used to derive wallet seeds.
