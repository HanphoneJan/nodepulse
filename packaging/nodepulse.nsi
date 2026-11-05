!include "MUI2.nsh"

!ifndef VERSION
  !define VERSION "0.3.5"
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

; DPI virtualization stretches the whole wizard as a bitmap at 125% and above.
ManifestDPIAware true
ManifestDPIAwareness "PerMonitorV2,System"
ManifestSupportedOS all

!define MUI_ABORTWARNING
!define MUI_ICON "${__FILEDIR__}\nodepulse.ico"
!define MUI_UNICON "${__FILEDIR__}\nodepulse.ico"
!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_BITMAP "${__FILEDIR__}\nodepulse-header.bmp"
!define MUI_HEADERIMAGE_BITMAP_STRETCH FitControl

!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "SimpChinese"

Section "NodePulse" SEC_MAIN
  SetShellVarContext current
  ; Stop a running instance so the installer can replace its files.
  nsExec::Exec 'taskkill /IM nodepulse.exe /F /T'
  Pop $0
  nsExec::Exec 'taskkill /IM nodepulse-core.exe /F /T'
  Pop $0
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
  nsExec::Exec 'taskkill /IM nodepulse.exe /F /T'
  Pop $0
  nsExec::Exec 'taskkill /IM nodepulse-core.exe /F /T'
  Pop $0
  Delete "$DESKTOP\NodePulse.lnk"
  Delete "$SMPROGRAMS\NodePulse\NodePulse.lnk"
  Delete "$SMPROGRAMS\NodePulse\卸载 NodePulse.lnk"
  RMDir "$SMPROGRAMS\NodePulse"
  RMDir /r "$INSTDIR"
SectionEnd

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SEC_MAIN} "安装 NodePulse 桌面端和本机后端。"
  !insertmacro MUI_DESCRIPTION_TEXT ${SEC_DESKTOP} "在桌面创建 NodePulse 快捷方式。"
!insertmacro MUI_FUNCTION_DESCRIPTION_END
