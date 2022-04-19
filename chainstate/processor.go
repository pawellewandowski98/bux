package chainstate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/mrz1836/go-whatsonchain"
	boom "github.com/tylertreat/BoomFilters"
)

// BloomProcessor bloom Filter processor
type BloomProcessor struct {
	debug             bool
	falsePositiveRate float64
	filters           map[string]*BloomProcessorFilter
	logger            Logger
	maxCells          uint
}

// BloomProcessorFilter struct
type BloomProcessorFilter struct {
	Filter *boom.StableBloomFilter
	regex  *regexp.Regexp
}

// NewBloomProcessor initialize a new bloom processor
func NewBloomProcessor(maxCells uint, falsePositiveRate float64) *BloomProcessor {
	return &BloomProcessor{
		filters:           make(map[string]*BloomProcessorFilter),
		maxCells:          maxCells,
		falsePositiveRate: falsePositiveRate,
	}
}

// Debug set debugging
func (p *BloomProcessor) Debug(debug bool) {
	p.debug = debug
}

// IsDebug return whether debugging is on/off
func (p *BloomProcessor) IsDebug() bool {
	return p.debug
}

// SetLogger set the logger
func (p *BloomProcessor) SetLogger(logger Logger) {
	p.logger = logger
}

// Logger return the logger
func (p *BloomProcessor) Logger() Logger {
	return p.logger
}

// GetHash get the hash of the current Filter
func (p *BloomProcessor) GetHash() string {
	h := sha256.New()
	for _, f := range p.filters {
		if _, err := f.Filter.WriteTo(h); err != nil {
			return ""
		}
	}
	hash := h.Sum(nil)
	return hex.EncodeToString(hash)
}

func (p *BloomProcessor) GetFilters() map[string]*BloomProcessorFilter {
	return p.filters
}

func (p *BloomProcessor) SetFilter(regex string, bloomFilter []byte) error {
	//p.filters[regex] =
	return nil
}

// Add a new item to the bloom Filter
func (p *BloomProcessor) Add(regexString, item string) error {

	// check whether this regex Filter already exists, otherwise initialize it
	if p.filters[regexString] == nil {
		r, err := regexp.Compile(regexString)
		if err != nil {
			return err
		}
		p.filters[regexString] = &BloomProcessorFilter{
			Filter: boom.NewDefaultStableBloomFilter(p.maxCells, p.falsePositiveRate),
			regex:  r,
		}
	}
	p.filters[regexString].Filter.Add([]byte(item))

	return nil
}

// Test checks whether the item is in the bloom Filter
func (p *BloomProcessor) Test(regexString, item string) bool {
	if p.filters[regexString] == nil {
		return false
	}
	return p.filters[regexString].Filter.Test([]byte(item))
}

// Reload the bloom Filter from the DB
func (p *BloomProcessor) Reload(regexString string, items []string) (err error) {
	for _, item := range items {
		if err = p.Add(regexString, item); err != nil {
			return
		}
	}

	return
}

// FilterTransactionPublishEvent check whether a Filter matches a tx event
func (p *BloomProcessor) FilterTransactionPublishEvent(eData []byte) (string, error) {
	transaction := whatsonchain.TxInfo{}
	_ = json.Unmarshal(eData, &transaction) // todo: missing error check

	for _, f := range p.filters {
		lockingScripts := f.regex.FindAllString(string(eData), -1)
		for _, ls := range lockingScripts {
			if passes := f.Filter.Test([]byte(ls)); passes {
				tx := whatsonchain.TxInfo{}
				if err := json.Unmarshal(eData, &tx); err != nil {
					return "", err
				}
				return tx.Hex, nil
			}
		}
	}

	return "", nil
}

// FilterTransaction check whether a Filter matches a tx event
func (p *BloomProcessor) FilterTransaction(txHex string) (string, error) {

	for _, f := range p.filters {
		lockingScripts := f.regex.FindAllString(txHex, -1)
		for _, ls := range lockingScripts {
			if passes := f.Filter.Test([]byte(ls)); passes {
				return txHex, nil
			}
		}
	}

	return "", nil
}

// RegexProcessor simple regex processor
// This processor just uses regex checks to see if a raw hex string exists in a tx
// This is bound to have some false positives but is somewhat performant when Filter set is small
type RegexProcessor struct {
	debug  bool
	filter []string
	logger Logger
}

// NewRegexProcessor initialize a new regex processor
func NewRegexProcessor() *RegexProcessor {
	return &RegexProcessor{
		filter: []string{},
	}
}

// Debug set debugging
func (p *RegexProcessor) Debug(debug bool) {
	p.debug = debug
}

// IsDebug return whether debugging is on/off
func (p *RegexProcessor) IsDebug() bool {
	return p.debug
}

// SetLogger set the logger
func (p *RegexProcessor) SetLogger(logger Logger) {
	p.logger = logger
}

// Logger return the logger
func (p *RegexProcessor) Logger() Logger {
	return p.logger
}

// Add a new item to the processor
func (p *RegexProcessor) Add(regex string, _ string) error {
	p.filter = append(p.filter, regex)
	return nil
}

// Test checks whether the item matches an item in the processor
func (p *RegexProcessor) Test(_ string, item string) bool {
	for _, f := range p.filter {
		if strings.Contains(item, f) {
			return true
		}
	}
	return false
}

// Reload the items of the processor to match against
func (p *RegexProcessor) Reload(_ string, items []string) (err error) {
	for _, i := range items {
		if err = p.Add(i, ""); err != nil {
			return
		}
	}
	return
}

// FilterTransactionPublishEvent check whether a Filter matches a tx event
func (p *RegexProcessor) FilterTransactionPublishEvent(eData []byte) (string, error) {
	transaction := whatsonchain.TxInfo{}
	if err := json.Unmarshal(eData, &transaction); err != nil {
		return "", err
	}
	if passes := p.Test("", transaction.Hex); !passes {
		return "", nil
	}
	return transaction.Hex, nil
}

// FilterTransaction filters transaction
func (p *RegexProcessor) FilterTransaction(hex string) (string, error) {
	if passes := p.Test("", hex); passes {
		return hex, nil
	}
	return "", nil
}

// GetHash get the hash of the Filter
func (p *RegexProcessor) GetHash() string {
	return ""
}
