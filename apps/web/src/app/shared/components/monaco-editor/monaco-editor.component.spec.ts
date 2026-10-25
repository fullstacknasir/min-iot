import { ComponentFixture, TestBed } from '@angular/core/testing';
import { FormsModule } from '@angular/forms';
import { MatIconModule } from '@angular/material/icon';
import { MonacoEditorComponent } from './monaco-editor.component';
import { ThemeService } from '../../../core/services/theme.service';

describe('MonacoEditorComponent', () => {
  let component: MonacoEditorComponent;
  let fixture: ComponentFixture<MonacoEditorComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [MonacoEditorComponent],
      imports: [FormsModule, MatIconModule],
      providers: [ThemeService],
    }).compileComponents();

    fixture = TestBed.createComponent(MonacoEditorComponent);
    component = fixture.componentInstance;
  });

  it('should initialize with default VS Code options', () => {
    component.code = '<div>test</div>';
    component.language = 'html';
    fixture.detectChanges();

    expect(component.language).toBe('html');
    expect(component.isWordWrapOn()).toBe(true);
    expect(component.isMinimapOn()).toBe(true);
  });

  it('should toggle word wrap, minimap, and theme', () => {
    fixture.detectChanges();

    component.toggleWordWrap();
    expect(component.isWordWrapOn()).toBe(false);

    component.toggleMinimap();
    expect(component.isMinimapOn()).toBe(false);

    const initialTheme = component.currentTheme();
    component.toggleTheme();
    expect(component.currentTheme()).not.toBe(initialTheme);
  });

  it('should emit codeChange on textarea input fallback', () => {
    fixture.detectChanges();

    let emitted = '';
    component.codeChange.subscribe((val) => {
      emitted = val;
    });

    component.onTextareaChange('self.onInit = function() {};');
    expect(emitted).toBe('self.onInit = function() {};');
    expect(component.code).toBe('self.onInit = function() {};');
  });
});
