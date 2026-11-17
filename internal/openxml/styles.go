package openxml

import "bytes"

// DefaultStylesXML returns a standard minimal styles.xml package for OpenXML spreadsheets.
// It includes styles for regular cells, bold header cells, green variance cells, and red variance cells.
func DefaultStylesXML() []byte {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
    <fonts count="2">
        <font>
            <sz val="11"/>
            <color theme="1"/>
            <name val="Calibri"/>
            <family val="2"/>
            <scheme val="minor"/>
        </font>
        <font>
            <b/>
            <sz val="11"/>
            <color theme="1"/>
            <name val="Calibri"/>
            <family val="2"/>
            <scheme val="minor"/>
        </font>
    </fonts>
    <fills count="4">
        <fill>
            <patternFill patternType="none"/>
        </fill>
        <fill>
            <patternFill patternType="gray125"/>
        </fill>
        <fill>
            <patternFill patternType="solid">
                <fgColor rgb="FFD4EDDA"/> <!-- Soft Green for variance <= 0 -->
                <bgColor indexed="64"/>
            </patternFill>
        </fill>
        <fill>
            <patternFill patternType="solid">
                <fgColor rgb="FFF8D7DA"/> <!-- Soft Red for variance > threshold -->
                <bgColor indexed="64"/>
            </patternFill>
        </fill>
    </fills>
    <borders count="2">
        <border>
            <left/><right/><top/><bottom/><diagonal/>
        </border>
        <border>
            <left style="thin"><color rgb="FFD3D3D3"/></left>
            <right style="thin"><color rgb="FFD3D3D3"/></right>
            <top style="thin"><color rgb="FFD3D3D3"/></top>
            <bottom style="thin"><color rgb="FFD3D3D3"/></bottom>
        </border>
    </borders>
    <cellStyleXfs count="1">
        <xf numFmtId="0" fontId="0" fillId="0" borderId="0"/>
    </cellStyleXfs>
    <cellXfs count="4">
        <!-- 0: Standard Default -->
        <xf numFmtId="0" fontId="0" fillId="0" borderId="1" xfId="0" applyFont="1" applyBorder="1"/>
        <!-- 1: Bold Header -->
        <xf numFmtId="0" fontId="1" fillId="0" borderId="1" xfId="0" applyFont="1" applyBorder="1"/>
        <!-- 2: Green Cell (Pass/Favorable) -->
        <xf numFmtId="0" fontId="0" fillId="2" borderId="1" xfId="0" applyFont="1" applyFill="1" applyBorder="1"/>
        <!-- 3: Red Cell (Alert/Discrepancy) -->
        <xf numFmtId="0" fontId="0" fillId="3" borderId="1" xfId="0" applyFont="1" applyFill="1" applyBorder="1"/>
    </cellXfs>
</styleSheet>`)
	return buf.Bytes()
}
