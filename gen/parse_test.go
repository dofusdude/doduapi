package gen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var testingLangs *map[string]LangDict
var testingData *JSONGameData

func setup() {
	testingLangs = ParseRawLanguages()
	testingData = ParseRawData()
}

func TestMain(m *testing.M) {
	setup()
	m.Run()
}

func TestParseSigness1(t *testing.T) {
	num, side := ParseSigness("-#1{~1~2 und -}#2")
	assert.True(t, num)
	assert.True(t, side)
}

func TestParseSigness2(t *testing.T) {
	num, side := ParseSigness("#1{~1~2 und -}#2")
	assert.False(t, num)
	assert.True(t, side)
}

func TestParseSigness3(t *testing.T) {
	num, side := ParseSigness("#1{~1~2 und }#2")
	assert.False(t, num)
	assert.False(t, side)
}

func TestParseSigness4(t *testing.T) {
	num, side := ParseSigness("-#1{~1~2 und }-#2")
	assert.True(t, num)
	assert.True(t, side)
}

func TestParseConditionSimple(t *testing.T) {
	condition := ParseCondition("cs<25", testingLangs, testingData)

	assert.Equal(t, 1, len(condition), "condition length")
	assert.Equal(t, "<", condition[0].Operator)
	assert.Equal(t, 25, condition[0].Value)
	assert.Equal(t, "Stärke", condition[0].Templated["de"])
}

func TestParseConditionMulti(t *testing.T) {
	condition := ParseCondition("CS>80&CV>40&CA>40", testingLangs, testingData)

	assert.Equal(t, len(condition), 3, "condition length")

	assert.Equal(t, ">", condition[0].Operator)
	assert.Equal(t, 80, condition[0].Value)
	assert.Equal(t, "Stärke", condition[0].Templated["de"])

	assert.Equal(t, ">", condition[1].Operator)
	assert.Equal(t, 40, condition[1].Value)
	assert.Equal(t, "Vitalität", condition[1].Templated["de"])

	assert.Equal(t, ">", condition[2].Operator)
	assert.Equal(t, 40, condition[2].Value)
	assert.Equal(t, "Flinkheit", condition[2].Templated["de"])
}

func TestDeleteNumHash(t *testing.T) {
	effect_name := DeleteDamageFormatter("Austauschbar ab: #1")
	assert.Equal(t, "Austauschbar ab:", effect_name)
}

func TestParseConditionEmpty(t *testing.T) {
	condition := ParseCondition("null", testingLangs, testingData)
	assert.Equal(t, 0, len(condition), "condition length")
}

func TestParseSingularPluralFormatterNormal(t *testing.T) {
	formatted := SingularPluralFormatter("Filzpunkte", 1, "de")
	assert.Equal(t, "Filzpunkte", formatted)
}

func TestParseSingularPluralFormatterPlural(t *testing.T) {
	formatted := SingularPluralFormatter("Kommt in %1 Subgebiet{~pen} vor", 2, "es")
	assert.Equal(t, "Kommt in %1 Subgebieten vor", formatted)

	formatted = SingularPluralFormatter("Punkt{~pe} erforderlich", 2, "es")
	assert.Equal(t, "Punkte erforderlich", formatted)
}

func TestParseSingularPluralFormatterPluralMulti(t *testing.T) {
	formatted := SingularPluralFormatter("Kommt in %1 Subgebiet{~pen} mit Punkt{~pe} vor", 2, "es")
	assert.Equal(t, "Kommt in %1 Subgebieten mit Punkte vor", formatted)
}

func TestParseSingularPluralFormatterSingularMulti(t *testing.T) {
	formatted := SingularPluralFormatter("Kommt in %1 Subgebiet{~sen} mit Punkt{~se} vor", 1, "es")
	assert.Equal(t, "Kommt in %1 Subgebieten mit Punkte vor", formatted)
}

func TestParseSingularPluralFormatterPluralComplexUnicode(t *testing.T) {
	formatted := SingularPluralFormatter("invocaç{~pões}", 2, "pt")
	assert.Equal(t, "invocações", formatted)
}

func TestParseSingularPluralFormatterPluralDeleteIfSingular(t *testing.T) {
	formatted := SingularPluralFormatter("invocaç{~pões}", 1, "pt")
	assert.Equal(t, "invocaç", formatted)
}

func TestDeleteDamageTemplate(t *testing.T) {
	formatted := DeleteDamageFormatter("#1{~1~2 bis }#2 (Erdschaden)")
	assert.Equal(t, "(Erdschaden)", formatted)
}

func TestDeleteDamageTemplateLevelEnBug(t *testing.T) {
	formatted := DeleteDamageFormatter("+#1{~1~2 to}level #2")
	assert.Equal(t, "level", formatted)
}

func TestParseNumSpellNameFormatterItSpecial(t *testing.T) {
	input := "Ottieni: #1{~1~2 - }#2 kama"
	diceNum := 100
	diceSide := 233
	value := 0
	output, _ := NumSpellFormatter(input, "it", testingData, testingLangs, &diceNum, &diceSide, &value, 0, false, false)

	assert.Equal(t, "Ottieni: 100 - 233 kama", output)
	assert.Equal(t, 100, diceNum)
	assert.Equal(t, 233, diceSide)
}

func TestParseNumSpellNameFormatterItSpecialSwitch(t *testing.T) {
	input := "#2: +#1 EP"
	diceNum := 100
	diceSide := 36
	value := 0
	output, _ := NumSpellFormatter(input, "it", testingData, testingLangs, &diceNum, &diceSide, &value, 0, false, false)

	assert.Equal(t, "36: +100 EP", output)
	assert.Equal(t, 100, diceNum)
	assert.Equal(t, 36, diceSide)
}

func TestParseNumSpellNameFormatterLearnSpellLevel(t *testing.T) {
	input := "Stufe #3 des Zauberspruchs erlernen"
	diceNum := 0
	diceSide := 0
	value := 1746
	output, _ := NumSpellFormatter(input, "de", testingData, testingLangs, &diceNum, &diceSide, &value, 0, false, false)

	assert.Equal(t, "Stufe 1746 des Zauberspruchs erlernen", output)
}

func TestParseNumSpellNameFormatterLearnSpellLevel1(t *testing.T) {
	input := "Stufe #3 des Zauberspruchs erlernen"
	diceNum := 0
	diceSide := 1
	value := 0
	output, _ := NumSpellFormatter(input, "de", testingData, testingLangs, &diceNum, &diceSide, &value, 0, false, false)

	assert.Equal(t, "Stufe 1 des Zauberspruchs erlernen", output)
}

func TestParseNumSpellNameFormatterDeNormal(t *testing.T) {
	input := "#1{~1~2 bis }#2 Kamagewinn"
	diceNum := 100
	diceSide := 233
	value := 0
	output, _ := NumSpellFormatter(input, "de", testingData, testingLangs, &diceNum, &diceSide, &value, 0, false, false)

	assert.Equal(t, "100 bis 233 Kamagewinn", output)
}

func TestParseNumSpellNameFormatterMultiValues(t *testing.T) {
	input := "Erfolgschance zwischen #1{~1~2 und }#2%"
	diceNum := 1
	diceSide := 2
	value := 0
	output, _ := NumSpellFormatter(input, "de", testingData, testingLangs, &diceNum, &diceSide, &value, 0, false, false)

	assert.Equal(t, "Erfolgschance zwischen 1 und 2%", output)

	input = "Erfolgschance zwischen -#1{~1~2 und -}#2%"
	diceNum = 1
	diceSide = 2
	value = 0
	output, _ = NumSpellFormatter(input, "de", testingData, testingLangs, &diceNum, &diceSide, &value, 0, false, false)

	assert.Equal(t, "Erfolgschance zwischen -1 und -2%", output)
}

func TestParseNumSpellNameFormatterVitaRange(t *testing.T) {
	input := "+#1{~1~2 bis }#2 Vitalität"
	diceNum := 0
	diceSide := 300
	value := 0
	output, _ := NumSpellFormatter(input, "de", testingData, testingLangs, &diceNum, &diceSide, &value, 0, false, true)

	assert.Equal(t, "+0 bis 300 Vitalität", output)
}

func TestParseNumSpellNameFormatterSingle(t *testing.T) {
	input := "Austauschbar ab: #1"
	diceNum := 1
	diceSide := 0
	value := 0
	output, _ := NumSpellFormatter(input, "de", testingData, testingLangs, &diceNum, &diceSide, &value, 0, false, false)

	assert.Equal(t, "Austauschbar ab: 1", output)
}

func TestParseNumSpellNameFormatterMinMax(t *testing.T) {
	input := "Verbleib. Anwendungen: #2 / #3" // delete the min max
	diceNum := 2
	diceSide := 5
	value := 6
	output, _ := NumSpellFormatter(input, "de", testingData, testingLangs, &diceNum, &diceSide, &value, 0, false, false)
	assert.Equal(t, "Verbleib. Anwendungen: 5 / 6", output)
}

func TestParseNumSpellNameFormatterSpellDiceNum(t *testing.T) {
	input := "Zauberwurf: #1"
	diceNum := 15960
	diceSide := 0
	value := 0
	output, _ := NumSpellFormatter(input, "de", testingData, testingLangs, &diceNum, &diceSide, &value, 0, false, false)

	assert.Equal(t, "Zauberwurf: Mauschelei", output)
}

func TestParseNumSpellNameFormatterEffectsRange(t *testing.T) {
	input := "-#1{~1~2 bis -}#2 Luftschaden"
	diceNum := 25
	diceSide := 50
	value := 0

	output, _ := NumSpellFormatter(input, "de", testingData, testingLangs, &diceNum, &diceSide, &value, 0, false, false)
	assert.Equal(t, "-25 bis -50 Luftschaden", output)
}

func TestParseNumSpellNameFormatterMissingWhite(t *testing.T) {
	input := "+#1{~1~2 to}level #2"
	diceNum := 1
	diceSide := 0
	value := 0

	output, _ := NumSpellFormatter(input, "de", testingData, testingLangs, &diceNum, &diceSide, &value, 0, false, false)
	assert.Equal(t, "+1 level", output)
}
