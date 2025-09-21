import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Eye, EyeOff, Building, User as UserIcon, Mail, Lock, UserPlus } from 'lucide-react';
import { authService } from '../../services/authService';

interface AuthFormProps {
  mode: 'login' | 'register';
  onModeSwitch: (mode: 'login' | 'register') => void;
}

interface RegisterData {
  name: string;
  username: string;
  email: string;
  password: string;
  confirmPassword: string;
  organizationMode: 'create' | 'join';
  organizationName?: string;
  organizationDescription?: string;
  organizationId?: string;
  allocatedSpaceMb?: number;
}

interface LoginData {
  email: string;
  password: string;
}

const AuthForm: React.FC<AuthFormProps> = ({ mode, onModeSwitch }) => {
  const navigate = useNavigate();
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>('');

  // Login form state
  const [loginData, setLoginData] = useState<LoginData>({
    email: '',
    password: '',
  });

  // Register form state
  const [registerData, setRegisterData] = useState<RegisterData>({
    name: '',
    username: '',
    email: '',
    password: '',
    confirmPassword: '',
    organizationMode: 'create',
    organizationName: '',
    organizationDescription: '',
    organizationId: '',
    allocatedSpaceMb: 100,
  });

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const response = await authService.login(loginData);
      
      // Store authentication data
      localStorage.setItem('token', response.tokenPair.access_token);
      localStorage.setItem('refreshToken', response.tokenPair.refresh_token);
      localStorage.setItem('user', JSON.stringify(response.user));
      localStorage.setItem('organization', JSON.stringify(response.organization));
      
      // Redirect to dashboard
      navigate('/dashboard');
    } catch (err: any) {
      setError(err.response?.data?.error || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    // Validation
    if (registerData.password !== registerData.confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    if (registerData.password.length < 6) {
      setError('Password must be at least 6 characters');
      return;
    }

    if (registerData.organizationMode === 'create' && !registerData.organizationName) {
      setError('Organization name is required');
      return;
    }

    if (registerData.organizationMode === 'join' && !registerData.organizationId) {
      setError('Organization ID is required');
      return;
    }

    setLoading(true);

    try {
      const payload = {
        name: registerData.name,
        username: registerData.username,
        email: registerData.email,
        password: registerData.password,
        ...(registerData.organizationMode === 'create' 
          ? {
              organizationName: registerData.organizationName,
              organizationDescription: registerData.organizationDescription,
              allocatedSpaceMb: registerData.allocatedSpaceMb,
            }
          : {
              organizationId: registerData.organizationId,
            }
        ),
      };

      const response = await authService.register(payload);
      
      // Store authentication data
      localStorage.setItem('token', response.tokenPair.access_token);
      localStorage.setItem('refreshToken', response.tokenPair.refresh_token);
      localStorage.setItem('user', JSON.stringify(response.user));
      localStorage.setItem('organization', JSON.stringify(response.organization));
      
      // Redirect to dashboard
      navigate('/dashboard');
    } catch (err: any) {
      setError(err.response?.data?.error || 'Registration failed');
    } finally {
      setLoading(false);
    }
  };

  const updateLoginData = (field: keyof LoginData, value: string) => {
    setLoginData(prev => ({ ...prev, [field]: value }));
  };

  const updateRegisterData = (field: keyof RegisterData, value: string | number) => {
    setRegisterData(prev => ({ ...prev, [field]: value }));
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center p-4">
      <div className="max-w-md w-full bg-white rounded-xl shadow-xl p-8">
        <div className="text-center mb-8">
          <div className="mx-auto w-16 h-16 bg-indigo-600 rounded-full flex items-center justify-center mb-4">
            <Building className="w-8 h-8 text-white" />
          </div>
          <h1 className="text-2xl font-bold text-gray-900">
            {mode === 'login' ? 'Welcome Back' : 'Create Account'}
          </h1>
          <p className="text-gray-600 mt-2">
            {mode === 'login' 
              ? 'Sign in to your secure file vault' 
              : 'Set up your organization and start managing files'
            }
          </p>
        </div>

        {error && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg">
            <p className="text-red-600 text-sm">{error}</p>
          </div>
        )}

        {mode === 'login' ? (
          <form onSubmit={handleLogin} className="space-y-6">
            {/* Email */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Email Address
              </label>
              <div className="relative">
                <Mail className="absolute left-3 top-3 h-5 w-5 text-gray-400" />
                <input
                  type="email"
                  required
                  className="pl-10 w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                  placeholder="you@example.com"
                  value={loginData.email}
                  onChange={(e) => updateLoginData('email', e.target.value)}
                />
              </div>
            </div>

            {/* Password */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Password
              </label>
              <div className="relative">
                <Lock className="absolute left-3 top-3 h-5 w-5 text-gray-400" />
                <input
                  type={showPassword ? 'text' : 'password'}
                  required
                  className="pl-10 pr-10 w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                  placeholder="Enter your password"
                  value={loginData.password}
                  onChange={(e) => updateLoginData('password', e.target.value)}
                />
                <button
                  type="button"
                  className="absolute right-3 top-3 text-gray-400 hover:text-gray-600"
                  onClick={() => setShowPassword(!showPassword)}
                >
                  {showPassword ? <EyeOff className="h-5 w-5" /> : <Eye className="h-5 w-5" />}
                </button>
              </div>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full bg-indigo-600 text-white p-3 rounded-lg hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed font-medium"
            >
              {loading ? 'Signing In...' : 'Sign In'}
            </button>
          </form>
        ) : (
          <form onSubmit={handleRegister} className="space-y-6">
            {/* Personal Information */}
            <div className="space-y-4">
              <h3 className="text-lg font-medium text-gray-900">Personal Information</h3>
              
              {/* Name */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Full Name
                </label>
                <div className="relative">
                  <UserIcon className="absolute left-3 top-3 h-5 w-5 text-gray-400" />
                  <input
                    type="text"
                    required
                    className="pl-10 w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                    placeholder="John Doe"
                    value={registerData.name}
                    onChange={(e) => updateRegisterData('name', e.target.value)}
                  />
                </div>
              </div>

              {/* Username */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Username
                </label>
                <div className="relative">
                  <UserPlus className="absolute left-3 top-3 h-5 w-5 text-gray-400" />
                  <input
                    type="text"
                    required
                    className="pl-10 w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                    placeholder="johndoe"
                    value={registerData.username}
                    onChange={(e) => updateRegisterData('username', e.target.value)}
                  />
                </div>
              </div>

              {/* Email */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Email Address
                </label>
                <div className="relative">
                  <Mail className="absolute left-3 top-3 h-5 w-5 text-gray-400" />
                  <input
                    type="email"
                    required
                    className="pl-10 w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                    placeholder="john@example.com"
                    value={registerData.email}
                    onChange={(e) => updateRegisterData('email', e.target.value)}
                  />
                </div>
              </div>

              {/* Password */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Password
                </label>
                <div className="relative">
                  <Lock className="absolute left-3 top-3 h-5 w-5 text-gray-400" />
                  <input
                    type={showPassword ? 'text' : 'password'}
                    required
                    className="pl-10 pr-10 w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                    placeholder="Create a password"
                    value={registerData.password}
                    onChange={(e) => updateRegisterData('password', e.target.value)}
                  />
                  <button
                    type="button"
                    className="absolute right-3 top-3 text-gray-400 hover:text-gray-600"
                    onClick={() => setShowPassword(!showPassword)}
                  >
                    {showPassword ? <EyeOff className="h-5 w-5" /> : <Eye className="h-5 w-5" />}
                  </button>
                </div>
              </div>

              {/* Confirm Password */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Confirm Password
                </label>
                <div className="relative">
                  <Lock className="absolute left-3 top-3 h-5 w-5 text-gray-400" />
                  <input
                    type={showConfirmPassword ? 'text' : 'password'}
                    required
                    className="pl-10 pr-10 w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                    placeholder="Confirm your password"
                    value={registerData.confirmPassword}
                    onChange={(e) => updateRegisterData('confirmPassword', e.target.value)}
                  />
                  <button
                    type="button"
                    className="absolute right-3 top-3 text-gray-400 hover:text-gray-600"
                    onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                  >
                    {showConfirmPassword ? <EyeOff className="h-5 w-5" /> : <Eye className="h-5 w-5" />}
                  </button>
                </div>
              </div>
            </div>

            {/* Organization Setup */}
            <div className="space-y-4">
              <h3 className="text-lg font-medium text-gray-900">Organization Setup</h3>
              
              {/* Organization Mode */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Organization
                </label>
                <div className="grid grid-cols-2 gap-2">
                  <button
                    type="button"
                    className={`p-3 text-sm font-medium border rounded-lg ${
                      registerData.organizationMode === 'create'
                        ? 'bg-indigo-50 border-indigo-300 text-indigo-700'
                        : 'bg-white border-gray-300 text-gray-700 hover:bg-gray-50'
                    }`}
                    onClick={() => updateRegisterData('organizationMode', 'create')}
                  >
                    Create New
                  </button>
                  <button
                    type="button"
                    className={`p-3 text-sm font-medium border rounded-lg ${
                      registerData.organizationMode === 'join'
                        ? 'bg-indigo-50 border-indigo-300 text-indigo-700'
                        : 'bg-white border-gray-300 text-gray-700 hover:bg-gray-50'
                    }`}
                    onClick={() => updateRegisterData('organizationMode', 'join')}
                  >
                    Join Existing
                  </button>
                </div>
              </div>

              {registerData.organizationMode === 'create' ? (
                <>
                  {/* Organization Name */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Organization Name
                    </label>
                    <div className="relative">
                      <Building className="absolute left-3 top-3 h-5 w-5 text-gray-400" />
                      <input
                        type="text"
                        required
                        className="pl-10 w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                        placeholder="Acme Corporation"
                        value={registerData.organizationName}
                        onChange={(e) => updateRegisterData('organizationName', e.target.value)}
                      />
                    </div>
                  </div>

                  {/* Organization Description */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Description (Optional)
                    </label>
                    <textarea
                      className="w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                      rows={2}
                      placeholder="Brief description of your organization"
                      value={registerData.organizationDescription}
                      onChange={(e) => updateRegisterData('organizationDescription', e.target.value)}
                    />
                  </div>

                  {/* Storage Allocation */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Initial Storage Allocation (MB)
                    </label>
                    <select
                      className="w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                      value={registerData.allocatedSpaceMb}
                      onChange={(e) => updateRegisterData('allocatedSpaceMb', parseInt(e.target.value))}
                    >
                      <option value={100}>100 MB (Demo)</option>
                      <option value={500}>500 MB</option>
                      <option value={1000}>1 GB</option>
                      <option value={5000}>5 GB</option>
                    </select>
                  </div>
                </>
              ) : (
                // Join existing organization
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Organization ID
                  </label>
                  <input
                    type="text"
                    required
                    className="w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                    placeholder="Organization UUID"
                    value={registerData.organizationId}
                    onChange={(e) => updateRegisterData('organizationId', e.target.value)}
                  />
                  <p className="text-xs text-gray-500 mt-1">
                    Get this ID from your organization administrator
                  </p>
                </div>
              )}
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full bg-indigo-600 text-white p-3 rounded-lg hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed font-medium"
            >
              {loading ? 'Creating Account...' : 'Create Account'}
            </button>
          </form>
        )}

        <div className="mt-6 text-center">
          <p className="text-gray-600">
            {mode === 'login' ? "Don't have an account?" : 'Already have an account?'}
            {' '}
            <button
              type="button"
              className="text-indigo-600 hover:text-indigo-500 font-medium"
              onClick={() => onModeSwitch(mode === 'login' ? 'register' : 'login')}
            >
              {mode === 'login' ? 'Sign up' : 'Sign in'}
            </button>
          </p>
        </div>
      </div>
    </div>
  );
};

export default AuthForm;