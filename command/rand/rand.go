package rand

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli"
	"go.step.sm/cli-utils/command"
	"go.step.sm/cli-utils/errs"
	"go.step.sm/crypto/fingerprint"
	"go.step.sm/crypto/randutil"
)

func init() {
	cmd := cli.Command{
		Name:      "rand",
		Action:    command.ActionFunc(randAction),
		Usage:     "generate random strings",
		UsageText: "**step rand** [<length>] [--format=<format>] [--dictionary=<file>]",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "format",
				Usage: "The <format> of the output string. See help for list of available formats.",
			},
			cli.StringFlag{
				Name:  "dictionary,dict",
				Usage: "The <file> to use as a dictionary to get random words.",
			},
		},
		Description: `**step rand** generates random strings that can be used for multiple purposes.
The <rand> command supports printing stings with different formats. It defaults
to use the printable characters of the ASCII table; <rand> also supports
generating a memorable password using words from a provided dictionary.

The length of the random string will default to 32 characters or 6 words
separated by a dash (-) if a dictionary is used.

The list of supported formats is the following:

* ascii: generates a random string using the 94 printable characters of the
  ASCII table.
* alphanumeric: uses the 62 alphanumeric characters in the POSIX/C locale
  (a-z+A-Z+0-9).
* alphabet: uses the 52 alphabetic characters in the POSIX/C locale (a-z+A-Z).
* hex: uses the 16 hexadecimal characters in lowercase (0-9+a-f).
* dec: uses the 10 decimal characters (0-9).
* lower: uses the 26 lowercase alphabetic characters in the POSIX/C locale
  (a-z).
* upper: uses the 26 uppercase alphabetic characters in the POSIX/C locale
  (A-Z).
* emoji: uses a curated list of 256 emojis that are not entirely similar.
* raw: uses random bytes.

The following special formats are also supported:

* dice: generates a random number between 1 and 6 or the given argument,
* uuid: generates a UUIDv4.

## POSITIONAL ARGUMENTS

<length>
:  The length of the random string in characters or words. If the dice format
is used, the length is the maximum number of the dice.

## EXAMPLES

Generate a random string using the default format (ascii) and length (32):
'''
$ step rand
Ijghm(Y?pfZiTPkHv0Z=1@MC<n&gsMe|
'''

Generate a random memorable string using a dictionary words:
'''
$ step rand --dictionary words.txt
scalpel-elan-fulsome-BELT-warring-balcony
'''

Generates a random roll of dice:
'''
$ step rand --format dice
4
'''

Generates a random hexadecimal string of 16 characters:
'''
$ step rand --format hex 16
f86a3f7b9299a413
'''

'''
Generates 20 upper-case characters:
$ step rand --format upper 20
LMCKDYUMRVJTTTZIKWGG
'''`}

	command.Register(cmd)
}

func randAction(ctx *cli.Context) error {
	if err := errs.MinMaxNumberOfArguments(ctx, 0, 1); err != nil {
		return err
	}

	var (
		err    error
		length int
	)

	dictionary := ctx.String("dictionary")
	format := strings.ToLower(ctx.String("format"))

	// Default to 32 characters, 6 words if a dictionary is used, or a dice roll
	// between 1 and 6.
	switch {
	case dictionary != "" && format != "":
		return errs.IncompatibleFlagWithFlag(ctx, "format", "dictionary")
	case dictionary != "", format == "dice":
		length = 6
	default:
		length = 32
	}

	if ctx.NArg() == 1 {
		arg := ctx.Args().First()
		length, err = strconv.Atoi(arg)
		if err != nil {
			return fmt.Errorf("positional argument <length> %q is not a valid number", arg)
		}
	}

	if dictionary != "" {
		return randWithDictionary(dictionary, length)
	}

	return randWithFormat(ctx, format, length)
}

func randWithFormat(ctx *cli.Context, format string, length int) error {
	var (
		s   string
		err error
	)

	if format == "raw" {
		b, err := randutil.Bytes(length)
		if err != nil {
			return err
		}
		os.Stdout.Write(b)
		return nil
	}

	switch format {
	case "", "ascii":
		s, err = randutil.ASCII(length)
	case "alphanumeric":
		s, err = randutil.Alphanumeric(length)
	case "alphabet":
		s, err = randutil.Alphabet(length)
	case "hex", "hexadecimal":
		s, err = randutil.Hex(length)
	case "dec", "decimal":
		s, err = randutil.String(length, "0123456789")
	case "lower":
		s, err = randutil.String(length, "abcdefghijklmnopqrstuvwxyz")
	case "upper":
		s, err = randutil.String(length, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	case "uuid":
		s, err = randutil.UUIDv4()
	case "emoji":
		b, err := randutil.Bytes(length)
		if err != nil {
			return err
		}
		s = fingerprint.Fingerprint(b, fingerprint.EmojiFingerprint)
	case "dice":
		bn, err := rand.Int(rand.Reader, big.NewInt(int64(length)))
		if err != nil {
			return err
		}
		s = bn.Add(bn, big.NewInt(1)).String()
	default:
		return errs.InvalidFlagValue(ctx, "format", format, "")
	}

	if err != nil {
		return err
	}

	fmt.Println(s)
	return nil
}

func randWithDictionary(dictionary string, length int) error {
	file, err := os.Open(dictionary)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	words := make([]string, 0, 1024)
	for scanner.Scan() {
		words = append(words, strings.TrimSpace(scanner.Text()))
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	var s string

	for i := 0; i < length; i++ {
		bn, err := rand.Int(rand.Reader, big.NewInt(int64(len(words))))
		if err != nil {
			return err
		}
		s += words[bn.Int64()]
		if i != length-1 {
			s += "-"
		}
	}

	fmt.Println(s)
	return nil
}
