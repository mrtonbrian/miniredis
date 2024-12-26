import socket

# Define the server and port
host = "127.0.0.1"
port = 6379

# Define the message in Redis RESP format
message = "*2\r\n$4\r\nECHO\r\n$3\r\nhey\r\n"

# Create a socket connection
with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
    # Connect to the Redis server
    s.connect((host, port))
    
    # Send the message
    s.sendall(message.encode('utf-8'))
    
    # Receive the response
    response = s.recv(1024)
    
    # Print the response
    print("Response from Redis:", response.decode('utf-8'))
