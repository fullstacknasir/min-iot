import {
  Component,
  Input,
  Output,
  EventEmitter,
  OnInit,
  OnDestroy,
  OnChanges,
  SimpleChanges,
  ElementRef,
  ViewChild,
  AfterViewInit,
  inject,
  signal,
} from '@angular/core';
import loader from '@monaco-editor/loader';
import type * as MonacoType from 'monaco-editor';
import { ThemeService } from '../../../core/services/theme.service';

const MONACO_ASSETS_PATH = 'assets/monaco/vs';
const MONACO_CDN_FALLBACK = 'https://cdn.jsdelivr.net/npm/monaco-editor@0.55.1/min/vs';
const MIME_JAVASCRIPT = 'javascript';
const DEFAULT_FONT_SIZE = 13;
const DEFAULT_TAB_SIZE = 2;
const THEME_DARK = 'vs-dark';
const THEME_LIGHT = 'vs';

export type MonacoSupportedLanguage = 'html' | 'css' | 'javascript' | 'json';

@Component({
  selector: 'app-monaco-editor',
  standalone: false,
  templateUrl: './monaco-editor.component.html',
})
export class MonacoEditorComponent implements OnInit, AfterViewInit, OnChanges, OnDestroy {
  @Input() public code = '';
  @Input() public language: MonacoSupportedLanguage = 'javascript';
  @Input() public readOnly = false;
  @Input() public minHeight = '520px';
  @Input() public showMinimap = true;

  @Output() public readonly codeChange = new EventEmitter<string>();

  @ViewChild('editorContainer', { static: false }) public editorContainer!: ElementRef<HTMLDivElement>;

  private readonly themeService = inject(ThemeService);

  private editorInstance?: MonacoType.editor.IStandaloneCodeEditor;
  private monacoApi?: typeof MonacoType;
  private isUpdatingInternally = false;
  private resizeObserver?: ResizeObserver;

  public readonly isLoaded = signal<boolean>(false);
  public readonly isInitializing = signal<boolean>(true);
  public readonly isWordWrapOn = signal<boolean>(true);
  public readonly isMinimapOn = signal<boolean>(true);
  public readonly currentTheme = signal<string>(THEME_DARK);

  public ngOnInit(): void {
    const isDark = this.themeService.isDarkMode();
    this.currentTheme.set(isDark ? THEME_DARK : THEME_LIGHT);
    this.isMinimapOn.set(this.showMinimap);
  }

  public ngAfterViewInit(): void {
    this.initMonacoInstance();
  }

  public ngOnChanges(changes: SimpleChanges): void {
    if (changes['code'] && !this.isUpdatingInternally && this.editorInstance) {
      const incoming = this.code ?? '';
      if (incoming !== this.editorInstance.getValue()) {
        this.editorInstance.setValue(incoming);
      }
    }

    if (changes['language'] && this.editorInstance && this.monacoApi) {
      const model = this.editorInstance.getModel();
      if (model) {
        this.monacoApi.editor.setModelLanguage(model, this.language);
      }
    }

    if (changes['readOnly'] && this.editorInstance) {
      this.editorInstance.updateOptions({ readOnly: this.readOnly });
    }
  }

  public ngOnDestroy(): void {
    this.resizeObserver?.disconnect();
    this.editorInstance?.dispose();
    this.editorInstance = undefined;
  }

  /**
   * Triggers VS Code document formatting (Shift+Alt+F)
   */
  public formatDocument(): void {
    if (!this.editorInstance) return;
    this.editorInstance.getAction('editor.action.formatDocument')?.run();
  }

  /**
   * Toggles line wrapping in the editor
   */
  public toggleWordWrap(): void {
    this.isWordWrapOn.update((v) => !v);
    this.editorInstance?.updateOptions({
      wordWrap: this.isWordWrapOn() ? 'on' : 'off',
    });
  }

  /**
   * Toggles code minimap visibility
   */
  public toggleMinimap(): void {
    this.isMinimapOn.update((v) => !v);
    this.editorInstance?.updateOptions({
      minimap: { enabled: this.isMinimapOn() },
    });
  }

  /**
   * Toggles editor theme between VS Code Dark+ and Light
   */
  public toggleTheme(): void {
    const next = this.currentTheme() === THEME_DARK ? THEME_LIGHT : THEME_DARK;
    this.currentTheme.set(next);
    if (this.monacoApi) {
      this.monacoApi.editor.setTheme(next);
    }
  }

  /**
   * Fallback textarea input handler when Monaco is unavailable
   */
  public onTextareaChange(val: string): void {
    this.code = val;
    this.codeChange.emit(val);
  }

  /**
   * Orchestrates Monaco Editor initialization with local & CDN auto-fallback
   */
  private async initMonacoInstance(): Promise<void> {
    if (typeof window === 'undefined' || !this.editorContainer?.nativeElement) {
      this.isInitializing.set(false);
      return;
    }

    try {
      const monaco = await this.attemptMonacoInit();
      this.onMonacoReady(monaco);
    } catch (err: unknown) {
      console.warn('Monaco initialization failed, rendering fallback:', err);
      this.isInitializing.set(false);
      this.isLoaded.set(false);
    }
  }

  /**
   * Attempts to load Monaco from local assets first, then falls back to CDN
   */
  private async attemptMonacoInit(): Promise<typeof MonacoType> {
    const hasLocal = await this.isLocalMonacoAvailable();
    const vsPath = hasLocal
      ? `${window.location.origin}/${MONACO_ASSETS_PATH}`
      : MONACO_CDN_FALLBACK;

    loader.config({ paths: { vs: vsPath } });

    try {
      return await loader.init();
    } catch (primaryErr) {
      if (hasLocal) {
        console.warn('Local Monaco load failed, switching to CDN fallback...', primaryErr);
        loader.config({ paths: { vs: MONACO_CDN_FALLBACK } });
        return await loader.init();
      }
      throw primaryErr;
    }
  }

  /**
   * Probes whether local self-hosted Monaco loader.js exists and is valid JS
   */
  private async isLocalMonacoAvailable(): Promise<boolean> {
    try {
      const probeUrl = `${window.location.origin}/${MONACO_ASSETS_PATH}/loader.js`;
      const response = await fetch(probeUrl, { method: 'HEAD' });
      const contentType = response.headers.get('content-type') ?? '';
      return response.ok && contentType.includes(MIME_JAVASCRIPT);
    } catch {
      return false;
    }
  }

  /**
   * Sets up editor components once Monaco instance is loaded
   */
  private onMonacoReady(monaco: typeof MonacoType): void {
    this.monacoApi = monaco;
    this.setupThingsBoardIntelliSense(monaco);
    this.createEditor(monaco);
    this.setupResizeObserver();
    this.isLoaded.set(true);
    this.isInitializing.set(false);
  }

  /**
   * Registers ThingsBoard JavaScript definitions for IntelliSense
   */
  private setupThingsBoardIntelliSense(monaco: typeof MonacoType): void {
    const languagesAny = monaco.languages as any;
    const jsDefaults = languagesAny?.typescript?.javascriptDefaults;
    if (!jsDefaults) return;

    jsDefaults.setDiagnosticsOptions({
      noSemanticValidation: false,
      noSyntaxValidation: false,
    });

    const thingsboardTypeDef = `
      interface ThingsBoardCtx {
        $container: HTMLElement;
        data: any;
        datasources: Array<{ deviceId: string; metricName?: string }>;
        settings: Record<string, any>;
        moment: any;
        echarts: any;
        currentUser?: { id: string; email: string; role: string };
      }
      interface ThingsBoardWidgetInstance {
        ctx: ThingsBoardCtx;
        onInit?: () => void;
        onDataUpdated?: (telemetry?: any) => void;
        onResize?: () => void;
        onDestroy?: () => void;
      }
      declare const self: ThingsBoardWidgetInstance;
      declare const ctx: ThingsBoardCtx;
      declare const moment: any;
      declare const echarts: any;
    `;

    try {
      jsDefaults.addExtraLib(thingsboardTypeDef, 'thingsboard.d.ts');
    } catch {
      // Ignored if already registered
    }
  }

  /**
   * Creates the Standalone Monaco Editor instance
   */
  private createEditor(monaco: typeof MonacoType): void {
    const domElement = this.editorContainer.nativeElement;
    domElement.innerHTML = '';

    const options: MonacoType.editor.IStandaloneEditorConstructionOptions = {
      value: this.code ?? '',
      language: this.language,
      theme: this.currentTheme(),
      automaticLayout: true,
      wordWrap: this.isWordWrapOn() ? 'on' : 'off',
      minimap: { enabled: this.isMinimapOn() },
      lineNumbers: 'on',
      scrollBeyondLastLine: false,
      tabSize: DEFAULT_TAB_SIZE,
      fontSize: DEFAULT_FONT_SIZE,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      cursorBlinking: 'smooth',
      formatOnPaste: true,
      formatOnType: true,
      readOnly: this.readOnly,
      padding: { top: 12, bottom: 12 },
    };

    this.editorInstance = monaco.editor.create(domElement, options);

    this.editorInstance.onDidChangeModelContent(() => {
      const val = this.editorInstance?.getValue() ?? '';
      this.isUpdatingInternally = true;
      this.code = val;
      this.codeChange.emit(val);
      this.isUpdatingInternally = false;
    });
  }

  /**
   * Observes container element to trigger layout resize
   */
  private setupResizeObserver(): void {
    if (typeof ResizeObserver === 'undefined') return;
    this.resizeObserver = new ResizeObserver(() => {
      this.editorInstance?.layout();
    });
    this.resizeObserver.observe(this.editorContainer.nativeElement);
  }
}
