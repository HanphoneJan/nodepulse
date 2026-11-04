!include "MUI2.nsh"

!ifndef VERSION
  !define VERSION "0.3.1"
!endif
!ifndef SOURCE
  !error "SOURCE is required"
!endif
!ifndef OUT
  !error "OUT is required"
!endif

Name "NodePulse"
OutFile "${OUT}"
InstallDir "$LOCALAPPDATA\Programs\NodePulse"
RequestExecutionLevel user
Unicode true

!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "SimpChinese"

Section "NodePulse" SEC_MAIN
  SetShellVarContext current
  SetOutPath "$INSTDIR"
  File /r "${SOURCE}\*"
  CreateDirectory "$SMPROGRAMS\NodePulse"
  CreateShortcut "$SMPROGRAMS\NodePulse\NodePulse.lnk" "$INSTDIR\nodepulse.exe" "" "$INSTDIR\nodepulse.ico"
  CreateShortcut "$SMPROGRAMS\NodePulse\卸载 NodePulse.lnk" "$INSTDIR\uninstall.exe"
  WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

Section "桌面快捷方式" SEC_DESKTOP
  SetShellVarContext current
  CreateShortcut "$DESKTOP\NodePulse.lnk" "$INSTDIR\nodepulse.exe" "" "$INSTDIR\nodepulse.ico"
SectionEnd

Section "Uninstall"
  SetShellVarContext current
  Delete "$DESKTOP\NodePulse.lnk"
  Delete "$SMPROGRAMS\NodePulse\NodePulse.lnk"
  Delete "$SMPROGRAMS\NodePulse\卸载 NodePulse.lnk"
  RMDir "$SMPROGRAMS\NodePulse"
  RMDir /r "$INSTDIR"
SectionEnd
