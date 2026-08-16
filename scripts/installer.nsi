; Linkquisition NSIS Installer Script
; Builds an installer that:
; - Installs the binary to Program Files
; - Registers as a URL handler (http/https)
; - Creates Start Menu shortcuts
; - Provides an uninstaller

!include "MUI2.nsh"
!include "FileFunc.nsh"

; --- Metadata ---
!define APPNAME "Linkquisition"
!define COMPANYNAME "Strobotti"
!define DESCRIPTION "A fast, configurable browser-picker"
!define VERSIONMAJOR 0
!define VERSIONMINOR 0
!define VERSIONBUILD 0
; VERSION is set at build time via /DVERSION=x.y.z
!ifndef VERSION
  !define VERSION "0.0.0"
!endif

!define INSTALLSIZE 35000 ; Approximate size in KB (includes Mesa OpenGL fallback)

; --- General ---
Name "${APPNAME}"
OutFile "dist\Linkquisition-${VERSION}-setup.exe"
InstallDir "$PROGRAMFILES\${APPNAME}"
InstallDirRegKey HKCU "Software\${APPNAME}" "InstallDir"
RequestExecutionLevel admin

; --- MUI Settings ---
!define MUI_ABORTWARNING
!define MUI_ICON "Icon.ico"

; --- Pages ---
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; --- Languages ---
!insertmacro MUI_LANGUAGE "English"

; --- Install Section ---
Section "Install"
    SetOutPath $INSTDIR

    ; Install the binary, icon, and Mesa OpenGL fallback
    File "dist\linkquisition.exe"
    File "Icon.ico"
    File "dist\opengl32.dll"
    File "NOTICE-MESA.txt"

    ; Store install directory
    WriteRegStr HKCU "Software\${APPNAME}" "InstallDir" "$INSTDIR"

    ; Create uninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"

    ; Start Menu shortcut (directly in Start Menu for Windows 10/11 visibility)
    CreateShortCut "$SMPROGRAMS\${APPNAME}.lnk" "$INSTDIR\linkquisition.exe" "" "$INSTDIR\Icon.ico"

    ; Register as URL handler
    ; -- URL Class
    WriteRegStr HKCU "Software\Classes\LinkquisitionURL" "" "${APPNAME} URL"
    WriteRegStr HKCU "Software\Classes\LinkquisitionURL" "URL Protocol" ""
    WriteRegStr HKCU "Software\Classes\LinkquisitionURL\shell\open\command" "" '"$INSTDIR\linkquisition.exe" "%1"'

    ; -- Capabilities
    WriteRegStr HKCU "Software\Clients\StartMenuInternet\${APPNAME}" "" "${APPNAME}"
    WriteRegStr HKCU "Software\Clients\StartMenuInternet\${APPNAME}\Capabilities" "ApplicationName" "${APPNAME}"
    WriteRegStr HKCU "Software\Clients\StartMenuInternet\${APPNAME}\Capabilities" "ApplicationDescription" "${DESCRIPTION}"
    WriteRegStr HKCU "Software\Clients\StartMenuInternet\${APPNAME}\Capabilities\URLAssociations" "http" "LinkquisitionURL"
    WriteRegStr HKCU "Software\Clients\StartMenuInternet\${APPNAME}\Capabilities\URLAssociations" "https" "LinkquisitionURL"
    WriteRegStr HKCU "Software\Clients\StartMenuInternet\${APPNAME}\shell\open\command" "" '"$INSTDIR\linkquisition.exe"'

    ; -- RegisteredApplications
    WriteRegStr HKCU "Software\RegisteredApplications" "${APPNAME}" "Software\Clients\StartMenuInternet\${APPNAME}\Capabilities"

    ; Add/Remove Programs entry
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "DisplayName" "${APPNAME}"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "UninstallString" '"$INSTDIR\uninstall.exe"'
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "DisplayIcon" "$INSTDIR\Icon.ico"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "InstallLocation" "$INSTDIR"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "Publisher" "${COMPANYNAME}"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "DisplayVersion" "${VERSION}"
    WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "NoModify" 1
    WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "NoRepair" 1
    WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "EstimatedSize" ${INSTALLSIZE}
SectionEnd

; --- Uninstall Section ---
Section "Uninstall"
    ; Remove files
    Delete "$INSTDIR\linkquisition.exe"
    Delete "$INSTDIR\Icon.ico"
    Delete "$INSTDIR\opengl32.dll"
    Delete "$INSTDIR\NOTICE-MESA.txt"
    Delete "$INSTDIR\uninstall.exe"
    RMDir "$INSTDIR"

    ; Remove Start Menu shortcut
    Delete "$SMPROGRAMS\${APPNAME}.lnk"

    ; Remove registry entries
    DeleteRegKey HKCU "Software\Classes\LinkquisitionURL"
    DeleteRegKey HKCU "Software\Clients\StartMenuInternet\${APPNAME}"
    DeleteRegValue HKCU "Software\RegisteredApplications" "${APPNAME}"
    DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}"
    DeleteRegKey HKCU "Software\${APPNAME}"
SectionEnd
