Unicode True
!include "MUI2.nsh"
!include "nsDialogs.nsh"
!include "LogicLib.nsh"
!include "x64.nsh"

Name "aFISHk"
OutFile "${OUTPUT}"
InstallDir "$LOCALAPPDATA\Programs\aFISHk"
RequestExecutionLevel user
SetCompressor /SOLID zlib
ShowInstDetails show
ShowUninstDetails show
BrandingText "aFISHk ${VERSION}"
Var StartupCheckbox
Var StartupEnabled

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "${STAGE}/LICENSE"
Page custom OptionsPage OptionsLeave
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_RUN
!define MUI_FINISHPAGE_RUN_FUNCTION Launch
!define MUI_FINISHPAGE_RUN_TEXT "Open aFISHk (first run opens browser setup)"
!define MUI_FINISHPAGE_RUN_NOTCHECKED
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Function .onInit
  SetShellVarContext current
  ${IfNot} ${IsNativeAMD64}
    MessageBox MB_ICONSTOP "This installer requires Windows 11 on an Intel/AMD 64-bit PC."
    Abort
  ${EndIf}
  SetRegView 64
  ReadRegStr $0 HKLM "SOFTWARE\Microsoft\Windows NT\CurrentVersion" "CurrentBuildNumber"
  ${If} $0 < 22000
    MessageBox MB_ICONSTOP "Windows 11 or later is required."
    Abort
  ${EndIf}
  StrCpy $StartupEnabled ${BST_UNCHECKED}
  ReadRegStr $0 HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "aFISHk"
  ${If} $0 != ""
    StrCpy $StartupEnabled ${BST_CHECKED}
  ${EndIf}
FunctionEnd

Function OptionsPage
  nsDialogs::Create 1018
  Pop $0
  ${If} $0 == error
    Abort
  ${EndIf}
  ${NSD_CreateLabel} 0 0 100% 48u "Installs for your account only. FFmpeg is included; no extra apps are needed.$\r$\nYou provide FishFry, your VPN and firewall settings.$\r$\nExisting settings and passwords are preserved on upgrades."
  Pop $0
  ${NSD_CreateCheckbox} 0 58u 100% 16u "Start aFISHk when I sign in"
  Pop $StartupCheckbox
  ${NSD_SetState} $StartupCheckbox $StartupEnabled
  nsDialogs::Show
FunctionEnd
Function OptionsLeave
  ${NSD_GetState} $StartupCheckbox $StartupEnabled
FunctionEnd

Section "Install"
  SetShellVarContext current
  SetRegView 64
  IfFileExists "$INSTDIR\afishk.exe" 0 stopped
    ExecWait '"$INSTDIR\afishk.exe" stop' $0
    ${If} $0 != 0
      MessageBox MB_ICONSTOP "Close aFISHk before upgrading, then run this installer again."
      Abort
    ${EndIf}
  stopped:
  SetOutPath "$INSTDIR"
  ClearErrors
  File "${STAGE}/afishk.exe"
  File "${STAGE}/ffmpeg.exe"
  File "${STAGE}/LICENSE"
  File "${STAGE}/README.txt"
  SetOutPath "$INSTDIR\licenses"
  File /r "${STAGE}/licenses/*"
  ${If} ${Errors}
    MessageBox MB_ICONSTOP "Could not install aFISHk. Close any running copy and try again."
    Abort
  ${EndIf}
  WriteUninstaller "$INSTDIR\Uninstall.exe"
  CreateDirectory "$SMPROGRAMS\aFISHk"
  CreateShortcut "$SMPROGRAMS\aFISHk\aFISHk.lnk" "$INSTDIR\afishk.exe" "launch"
  CreateShortcut "$SMPROGRAMS\aFISHk\Stop aFISHk.lnk" "$INSTDIR\afishk.exe" "stop"
  CreateShortcut "$SMPROGRAMS\aFISHk\Uninstall aFISHk.lnk" "$INSTDIR\Uninstall.exe"
  CreateShortcut "$SMPROGRAMS\aFISHk\Settings folder.lnk" "$APPDATA\afishk"
  CreateShortcut "$SMPROGRAMS\aFISHk\Logs folder.lnk" "$LOCALAPPDATA\afishk"
  ${If} $StartupEnabled == ${BST_CHECKED}
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "aFISHk" '"$INSTDIR\afishk.exe" launch --background'
  ${Else}
    DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "aFISHk"
  ${EndIf}
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\aFISHk" "DisplayName" "aFISHk"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\aFISHk" "DisplayVersion" "${VERSION}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\aFISHk" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\aFISHk" "UninstallString" '"$INSTDIR\Uninstall.exe"'
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\aFISHk" "NoModify" 1
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\aFISHk" "NoRepair" 1
SectionEnd

Function Launch
  Exec '"$INSTDIR\afishk.exe" launch'
FunctionEnd

Section "Uninstall"
  SetShellVarContext current
  SetRegView 64
  ExecWait '"$INSTDIR\afishk.exe" stop' $0
  ${If} $0 != 0
    MessageBox MB_ICONSTOP "Close aFISHk before uninstalling, then try again."
    Abort
  ${EndIf}
  DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "aFISHk"
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\aFISHk"
  Delete "$SMPROGRAMS\aFISHk\aFISHk.lnk"
  Delete "$SMPROGRAMS\aFISHk\Stop aFISHk.lnk"
  Delete "$SMPROGRAMS\aFISHk\Uninstall aFISHk.lnk"
  Delete "$SMPROGRAMS\aFISHk\Settings folder.lnk"
  Delete "$SMPROGRAMS\aFISHk\Logs folder.lnk"
  RMDir "$SMPROGRAMS\aFISHk"
  Delete "$INSTDIR\afishk.exe"
  Delete "$INSTDIR\ffmpeg.exe"
  Delete "$INSTDIR\LICENSE"
  Delete "$INSTDIR\README.txt"
  Delete "$INSTDIR\licenses\FFmpeg-LGPL-2.1.txt"
  Delete "$INSTDIR\licenses\Opus-BSD.txt"
  Delete "$INSTDIR\licenses\MinGW-copyright.txt"
  Delete "$INSTDIR\licenses\GCC-runtime-copyright.txt"
  Delete "$INSTDIR\licenses\GPL-3.txt"
  Delete "$INSTDIR\licenses\NSIS-copyright.txt"
  Delete "$INSTDIR\licenses\NOTICE.txt"
  RMDir "$INSTDIR\licenses"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"
  ; Deliberately preserve APPDATA config/token and LOCALAPPDATA cache/logs.
SectionEnd
