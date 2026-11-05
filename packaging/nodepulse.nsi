!include "MUI2.nsh"
!include "Sections.nsh"

!ifndef VERSION
  !define VERSION "0.3.6"
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
; Reuse the path chosen by the previous install, so upgrades start in place.
InstallDirRegKey HKCU "Software\NodePulse" "InstallDir"
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

!define MUI_PAGE_CUSTOMFUNCTION_PRE npDirectoryPagePre
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

  ; Remember the chosen directory for the next upgrade.
  WriteRegStr HKCU "Software\NodePulse" "InstallDir" "$INSTDIR"

  ; Add/Remove Programs entry, also used to locate the previous installation.
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\NodePulse" "DisplayName" "NodePulse"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\NodePulse" "DisplayVersion" "${VERSION}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\NodePulse" "Publisher" "NodePulse"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\NodePulse" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\NodePulse" "DisplayIcon" "$INSTDIR\nodepulse.exe,0"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\NodePulse" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\NodePulse" "NoModify" 1
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\NodePulse" "NoRepair" 1
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
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\NodePulse"
  DeleteRegKey HKCU "Software\NodePulse"
  RMDir /r "$INSTDIR"
SectionEnd

; NodePulse 0.3.6 and earlier did not persist this choice, so a missing value
; means "fresh install" and keeps the default selected state.
Function .onInit
  SetShellVarContext current
  ClearErrors
  ReadRegDWORD $0 HKCU "Software\NodePulse" "DesktopShortcut"
  IfErrors np_desktop_init_done
  StrCmp $0 "0" np_desktop_init_off
  SectionSetFlags ${SEC_DESKTOP} ${SF_SELECTED}
  Goto np_desktop_init_done
np_desktop_init_off:
  SectionSetFlags ${SEC_DESKTOP} 0
np_desktop_init_done:
FunctionEnd

; An existing InstallDir means this is an upgrade. Install in place and skip
; the directory page; a fresh install still shows it.
Function npDirectoryPagePre
  ReadRegStr $0 HKCU "Software\NodePulse" "InstallDir"
  StrCmp $0 "" np_directory_show
  Abort
np_directory_show:
FunctionEnd

Function .onInstSuccess
  SetShellVarContext current
  SectionGetFlags ${SEC_DESKTOP} $0
  IntOp $0 $0 & ${SF_SELECTED}
  StrCmp $0 ${SF_SELECTED} np_desktop_selected
  WriteRegDWORD HKCU "Software\NodePulse" "DesktopShortcut" 0
  ; Also remove a shortcut left by an earlier version.
  Delete "$DESKTOP\NodePulse.lnk"
  Goto np_desktop_saved
np_desktop_selected:
  WriteRegDWORD HKCU "Software\NodePulse" "DesktopShortcut" 1
np_desktop_saved:
FunctionEnd

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SEC_MAIN} "安装 NodePulse 桌面端和本机后端。"
  !insertmacro MUI_DESCRIPTION_TEXT ${SEC_DESKTOP} "在桌面创建 NodePulse 快捷方式。"
!insertmacro MUI_FUNCTION_DESCRIPTION_END
