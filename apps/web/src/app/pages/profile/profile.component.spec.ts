import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { provideNoopAnimations } from '@angular/platform-browser/animations';
import { of } from 'rxjs';
import { AppModule } from '../../app.module';
import { ProfileComponent } from './profile.component';
import { AuthService } from '../../core/services/auth.service';

describe('ProfileComponent', () => {
  let authService: AuthService;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AppModule],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideNoopAnimations(),
      ],
    }).compileComponents();

    authService = TestBed.inject(AuthService);
  });

  it('should initialize and load user profile form', () => {
    const mockUser = {
      id: 'usr-123',
      organization_id: 'org-456',
      email: 'test@example.com',
      name: 'Test Engineer',
      role: 'operator' as const,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };

    vi.spyOn(authService, 'getProfile').mockReturnValue(
      of({ user: mockUser, organization: { id: 'org-456', name: 'Test Org', created_at: '', updated_at: '' } })
    );

    const fixture = TestBed.createComponent(ProfileComponent);
    const component = fixture.componentInstance;
    fixture.detectChanges();

    expect(component.user()?.email).toBe('test@example.com');
    expect(component.editName).toBe('Test Engineer');
    expect(component.editEmail).toBe('test@example.com');
  });

  it('should update profile when save is triggered', () => {
    const updatedUser = {
      id: 'usr-123',
      organization_id: 'org-456',
      email: 'updated@example.com',
      name: 'Updated Name',
      role: 'operator' as const,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };

    vi.spyOn(authService, 'getProfile').mockReturnValue(
      of({ user: updatedUser, organization: undefined })
    );
    vi.spyOn(authService, 'updateProfile').mockReturnValue(
      of({ user: updatedUser, organization: undefined })
    );

    const fixture = TestBed.createComponent(ProfileComponent);
    const component = fixture.componentInstance;
    fixture.detectChanges();

    component.editName = 'Updated Name';
    component.editEmail = 'updated@example.com';
    component.handleSaveProfile();

    expect(authService.updateProfile).toHaveBeenCalledWith({
      name: 'Updated Name',
      email: 'updated@example.com',
    });
    expect(component.successMessage()).toBe('Profile updated successfully.');
  });
});
