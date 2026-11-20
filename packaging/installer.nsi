Unicode true
!include "MUI2.nsh"
!include "x64.nsh"
!include "WinVer.nsh"
!include "LogicLib.nsh"

!ifndef APP_BINARY
!define APP_BINARY "..\build\bin\elastic-ui-release.exe"
!endif
!ifndef APP_VERSION
!define APP_VERSION "1.0.0"
!endif
!define APP_NAME "Elastic Defend 本机控制台"
!ifndef UNINSTALL_KEY
!define UNINSTALL_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\ElasticDefendConsole"
!endif
!ifndef OUTPUT_FILE
!define OUTPUT_FILE "..\build\bin\elastic-ui-windows-amd64-setup.exe"
!endif
!ifndef NO_SHORTCUTS
!define CREATE_SHORTCUTS
!endif

Name "${APP_NAME}"
OutFile "${OUTPUT_FILE}"
InstallDir "$PROGRAMFILES64\Elastic UI"
InstallDirRegKey HKLM "${UNINSTALL_KEY}" "InstallLocation"
RequestExecutionLevel admin
ManifestDPIAware true
SetCompressor /SOLID lzma
VIProductVersion "${APP_VERSION}.0"
VIAddVersionKey "ProductName" "${APP_NAME}"
VIAddVersionKey "FileDescription" "Elastic UI Installer"
VIAddVersionKey "FileVersion" "${APP_VERSION}"
VIAddVersionKey "ProductVersion" "${APP_VERSION}"
VIAddVersionKey "LegalCopyright" "Elastic UI contributors"

!define MUI_ICON "..\build\windows\icon.ico"
!define MUI_UNICON "..\build\windows\icon.ico"
!define MUI_ABORTWARNING
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_RUN "$INSTDIR\elastic-ui.exe"
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "SimpChinese"
!insertmacro MUI_LANGUAGE "English"

Function .onInit
  ${IfNot} ${AtLeastWin10}
    MessageBox MB_OK "需要 Windows 10 或更新版本。"
    Abort
  ${EndIf}
  ${IfNot} ${RunningX64}
    MessageBox MB_OK "需要 64 位 Windows。"
    Abort
  ${EndIf}
  SetRegView 64
!ifndef TEST_INSTALL
  ; Do not rename a running executable: the old process would continue running
  ; and the restarted copy would exit due to the single-instance mutex.
  System::Call 'kernel32::OpenMutexW(i 0x100000, i 0, w "Local\ElasticDefendConsoleMutex") p .r0'
  ${If} $0 != 0
    System::Call 'kernel32::CloseHandle(p r0)'
    MessageBox MB_OK "请先退出 Elastic UI，再运行安装程序。" /SD IDOK
    SetErrorLevel 12
    Abort
  ${EndIf}
!endif
FunctionEnd

Section "安装"
  SetShellVarContext all
  SetRegView 32
  ReadRegStr $0 HKLM "SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" "pv"
  ${If} $0 == ""
    ReadRegStr $0 HKCU "Software\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" "pv"
  ${EndIf}
  SetRegView 64
  ${If} $0 == ""
    InitPluginsDir
    SetOutPath "$PLUGINSDIR"
    File "MicrosoftEdgeWebview2Setup.exe"
    ExecWait '"$PLUGINSDIR\MicrosoftEdgeWebview2Setup.exe" /silent /install' $0
    ${If} $0 != 0
      SetErrorLevel 10
      Abort "WebView2 安装失败。"
    ${EndIf}
  ${EndIf}

  SetOutPath "$INSTDIR"
  ClearErrors
  File /oname=elastic-ui.new.exe "${APP_BINARY}"
  IfErrors install_failed
  Delete "$INSTDIR\elastic-ui.previous.exe"
  IfFileExists "$INSTDIR\elastic-ui.exe" 0 replace_new
  ClearErrors
  Rename "$INSTDIR\elastic-ui.exe" "$INSTDIR\elastic-ui.previous.exe"
  IfErrors install_failed
replace_new:
  ClearErrors
  Rename "$INSTDIR\elastic-ui.new.exe" "$INSTDIR\elastic-ui.exe"
  IfErrors rollback
  WriteUninstaller "$INSTDIR\uninstall.exe"
!ifdef CREATE_SHORTCUTS
  CreateShortcut "$SMPROGRAMS\${APP_NAME}.lnk" "$INSTDIR\elastic-ui.exe"
  CreateShortcut "$DESKTOP\${APP_NAME}.lnk" "$INSTDIR\elastic-ui.exe"
!endif
  WriteRegStr HKLM "${UNINSTALL_KEY}" "DisplayName" "${APP_NAME}"
  WriteRegStr HKLM "${UNINSTALL_KEY}" "DisplayVersion" "${APP_VERSION}"
  WriteRegStr HKLM "${UNINSTALL_KEY}" "Publisher" "Elastic UI contributors"
  WriteRegStr HKLM "${UNINSTALL_KEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKLM "${UNINSTALL_KEY}" "DisplayIcon" "$INSTDIR\elastic-ui.exe"
  WriteRegStr HKLM "${UNINSTALL_KEY}" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegStr HKLM "${UNINSTALL_KEY}" "QuietUninstallString" '"$INSTDIR\uninstall.exe" /S'
  WriteRegDWORD HKLM "${UNINSTALL_KEY}" "NoModify" 1
  WriteRegDWORD HKLM "${UNINSTALL_KEY}" "NoRepair" 1
!ifndef TEST_INSTALL
  ExecWait '"$INSTDIR\elastic-ui.exe" --refresh-integration'
!endif
  SetErrorLevel 0
  Goto done
rollback:
  Rename "$INSTDIR\elastic-ui.previous.exe" "$INSTDIR\elastic-ui.exe"
install_failed:
  Delete "$INSTDIR\elastic-ui.new.exe"
  SetErrorLevel 11
  Abort "无法替换程序。请退出 Elastic UI 后重试。"
done:
SectionEnd

Section "Uninstall"
  SetShellVarContext all
  SetRegView 64
!ifndef TEST_INSTALL
  ExecWait '"$INSTDIR\elastic-ui.exe" --uninstall-integration'
!endif
  ClearErrors
  Delete "$INSTDIR\elastic-ui.exe"
  IfErrors 0 +3
    MessageBox MB_OK "请退出 Elastic UI 后重试卸载。"
    Abort
  Delete "$INSTDIR\elastic-ui.previous.exe"
  Delete "$INSTDIR\uninstall.exe"
!ifdef CREATE_SHORTCUTS
  Delete "$SMPROGRAMS\${APP_NAME}.lnk"
  Delete "$DESKTOP\${APP_NAME}.lnk"
!endif
  DeleteRegKey HKLM "${UNINSTALL_KEY}"
  RMDir "$INSTDIR"
SectionEnd
