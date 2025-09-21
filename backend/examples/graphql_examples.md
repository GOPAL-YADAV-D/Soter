# Example GraphQL Mutations and Queries for File Upload System

## 1. Create Upload Session
```graphql
mutation CreateUploadSession($input: UploadSessionInput!) {
  createUploadSession(input: $input) {
    id
    sessionToken
    totalFiles
    totalBytes
    status
    startedAt
  }
}
```

**Variables:**
```json
{
  "input": {
    "files": [
      {
        "filename": "document.pdf",
        "mimeType": "application/pdf",
        "fileSize": 1048576,
        "folderPath": "/documents",
        "contentHash": "a1b2c3d4e5f6..."
      },
      {
        "filename": "image.jpg",
        "mimeType": "image/jpeg",
        "fileSize": 2097152,
        "folderPath": "/images"
      }
    ],
    "totalBytes": 3145728
  }
}
```

## 2. Query Upload Progress
```graphql
query GetUploadProgress($sessionToken: String!) {
  uploadProgress(sessionToken: $sessionToken) {
    sessionID
    sessionToken
    totalFiles
    completedFiles
    failedFiles
    totalBytes
    uploadedBytes
    status
    progressPercent
  }
}
```

**Variables:**
```json
{
  "sessionToken": "session-token-uuid-here"
}
```

## 3. Query User Files
```graphql
query GetUserFiles($folderPath: String, $limit: Int, $offset: Int) {
  userFiles(folderPath: $folderPath, limit: $limit, offset: $offset) {
    id
    userFilename
    uploadDate
    folderPath
    file {
      id
      contentHash
      filename
      originalMimeType
      detectedMimeType
      fileSize
      uploadDate
    }
  }
}
```

**Variables:**
```json
{
  "folderPath": "/documents",
  "limit": 20,
  "offset": 0
}
```

## 4. Query Storage Statistics
```graphql
query GetStorageStats {
  storageStats {
    id
    totalFiles
    uniqueFiles
    totalSizeBytes
    actualStorageBytes
    savingsBytes
    savingsPercentage
    lastCalculated
  }
}
```

## 5. Query Current User with Storage Stats
```graphql
query GetMe {
  me {
    id
    email
    firstName
    lastName
    isActive
    createdAt
    storageStats {
      totalFiles
      uniqueFiles
      totalSizeBytes
      actualStorageBytes
      savingsBytes
      savingsPercentage
    }
  }
}
```

## 6. Complete Upload Session
```graphql
mutation CompleteUploadSession($sessionToken: String!) {
  completeUploadSession(sessionToken: $sessionToken)
}
```

**Variables:**
```json
{
  "sessionToken": "session-token-uuid-here"
}
```

## 7. Delete File
```graphql
mutation DeleteFile($userFileID: UUID!) {
  deleteFile(userFileID: $userFileID)
}
```

**Variables:**
```json
{
  "userFileID": "file-uuid-here"
}
```

## 8. Authentication Mutations
```graphql
mutation Register($input: RegisterInput!) {
  register(input: $input) {
    token
    user {
      id
      email
      firstName
      lastName
      isActive
      createdAt
    }
  }
}
```

**Variables:**
```json
{
  "input": {
    "email": "user@example.com",
    "password": "securepassword",
    "firstName": "John",
    "lastName": "Doe"
  }
}
```

```graphql
mutation Login($input: LoginInput!) {
  login(input: $input) {
    token
    user {
      id
      email
      firstName
      lastName
      storageStats {
        totalFiles
        savingsBytes
        savingsPercentage
      }
    }
  }
}
```

**Variables:**
```json
{
  "input": {
    "email": "user@example.com",
    "password": "securepassword"
  }
}
```

## Example Responses

### Upload Session Response:
```json
{
  "data": {
    "createUploadSession": {
      "id": "session-uuid",
      "sessionToken": "session-token-uuid",
      "totalFiles": 2,
      "totalBytes": 3145728,
      "status": "pending",
      "startedAt": "2025-09-20T10:30:00Z"
    }
  }
}
```

### Upload Progress Response:
```json
{
  "data": {
    "uploadProgress": {
      "sessionID": "session-uuid",
      "sessionToken": "session-token-uuid",
      "totalFiles": 2,
      "completedFiles": 1,
      "failedFiles": 0,
      "totalBytes": 3145728,
      "uploadedBytes": 1048576,
      "status": "in_progress",
      "progressPercent": 33.33
    }
  }
}
```

### Storage Statistics Response:
```json
{
  "data": {
    "storageStats": {
      "id": "stats-uuid",
      "totalFiles": 150,
      "uniqueFiles": 120,
      "totalSizeBytes": 10737418240,
      "actualStorageBytes": 8589934592,
      "savingsBytes": 2147483648,
      "savingsPercentage": 20.00,
      "lastCalculated": "2025-09-20T10:30:00Z"
    }
  }
}
```

### User Files Response:
```json
{
  "data": {
    "userFiles": [
      {
        "id": "userfile-uuid-1",
        "userFilename": "my-document.pdf",
        "uploadDate": "2025-09-20T09:00:00Z",
        "folderPath": "/documents",
        "file": {
          "id": "file-uuid-1",
          "contentHash": "a1b2c3d4e5f6...",
          "filename": "document.pdf",
          "originalMimeType": "application/pdf",
          "detectedMimeType": "application/pdf",
          "fileSize": 1048576,
          "uploadDate": "2025-09-20T09:00:00Z"
        }
      }
    ]
  }
}
```